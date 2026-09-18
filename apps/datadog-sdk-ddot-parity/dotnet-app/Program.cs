using System.Diagnostics;
using System.Runtime.CompilerServices;
using Npgsql;

var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();

var serviceName = Environment.GetEnvironmentVariable("OTEL_SERVICE_NAME")
    ?? Environment.GetEnvironmentVariable("DD_SERVICE")
    ?? "otel-dotnet-ddot-parity";
var activitySource = new ActivitySource("otel-dotnet-validation");
var retainedAllocations = new Queue<byte[]>();

string ConnectionString()
{
    var host = Environment.GetEnvironmentVariable("POSTGRES_HOST") ?? "postgres";
    var port = Environment.GetEnvironmentVariable("POSTGRES_DB_PORT") ?? "5432";
    var database = Environment.GetEnvironmentVariable("POSTGRES_DB") ?? "validation";
    var username = Environment.GetEnvironmentVariable("POSTGRES_USER") ?? "validation";
    var password = Environment.GetEnvironmentVariable("POSTGRES_PASSWORD") ?? "validation";
    return $"Host={host};Port={port};Database={database};Username={username};Password={password};Application Name={serviceName}";
}

app.MapGet("/health", () => Results.Json(new
{
    status = "ok",
    runtime = Environment.Version.ToString(),
    service = serviceName,
    time = DateTimeOffset.UtcNow,
}));

app.MapGet("/span", () =>
{
    using var activity = activitySource.StartActivity("manual.activity.bridge", ActivityKind.Internal);
    activity?.SetTag("validation.bridge", "activity-source");
    activity?.SetTag("validation.sdk", "dd-trace-dotnet-3.53.0");
    Thread.Sleep(100);
    return Results.Json(new { span = "ok", activity = activity?.Id });
});

app.MapGet("/probe", (int value) =>
{
    var result = ProbeTargets.DynamicProbeTarget(value);
    return Results.Json(new { input = value, result });
});

app.MapGet("/cpu", (double seconds = 1.5) =>
{
    seconds = Math.Clamp(seconds, 0.1, 10.0);
    var stopwatch = Stopwatch.StartNew();
    long iterations = 0;
    long checksum = 0;
    while (stopwatch.Elapsed.TotalSeconds < seconds)
    {
        checksum = (checksum + (iterations * 31) % 9973) % 1_000_003;
        iterations++;
    }

    return Results.Json(new { seconds, iterations, checksum });
});

app.MapGet("/allocate", (int mb = 8) =>
{
    mb = Math.Clamp(mb, 1, 64);
    var bytes = new byte[mb * 1024 * 1024];
    bytes[0] = 1;
    bytes[^1] = 1;
    retainedAllocations.Enqueue(bytes);
    while (retainedAllocations.Count > 4)
    {
        retainedAllocations.Dequeue();
    }

    return Results.Json(new { allocatedMb = mb, retainedBlocks = retainedAllocations.Count });
});

app.MapGet("/exceptions", (int count = 250) =>
{
    count = Math.Clamp(count, 1, 5_000);
    var caught = 0;
    for (var index = 0; index < count; index++)
    {
        try
        {
            throw new ValidationProfileException($"intentional-profile-exception-{index % 5}");
        }
        catch (ValidationProfileException)
        {
            caught++;
        }
    }

    return Results.Json(new { caught });
});

app.MapGet("/db/query", async () =>
{
    await using var connection = new NpgsqlConnection(ConnectionString());
    await connection.OpenAsync();
    await using var command = new NpgsqlCommand(
        "SELECT status, count(*) FROM orders WHERE customer_id = @customer_id GROUP BY status",
        connection);
    command.Parameters.AddWithValue("customer_id", 42);
    await using var reader = await command.ExecuteReaderAsync();
    var rows = new List<object>();
    while (await reader.ReadAsync())
    {
        rows.Add(new { status = reader.GetString(0), count = reader.GetInt64(1) });
    }

    return Results.Json(new { rows });
});

app.MapGet("/db/slow", async () =>
{
    await using var connection = new NpgsqlConnection(ConnectionString());
    await connection.OpenAsync();
    await using var command = new NpgsqlCommand("SELECT pg_sleep(1.25), count(*) FROM orders", connection);
    await using var reader = await command.ExecuteReaderAsync();
    await reader.ReadAsync();
    return Results.Json(new { count = reader.GetInt64(1), sleptSeconds = 1.25 });
});

app.MapGet("/db/error", async () =>
{
    try
    {
        await using var connection = new NpgsqlConnection(ConnectionString());
        await connection.OpenAsync();
        await using var command = new NpgsqlCommand("SELECT * FROM validation_table_that_does_not_exist", connection);
        await command.ExecuteNonQueryAsync();
    }
    catch (PostgresException exception)
    {
        return Results.Json(new { error = exception.SqlState }, statusCode: 500);
    }

    return Results.Json(new { error = "not raised" }, statusCode: 500);
});

app.Run("http://0.0.0.0:5000");

public static class ProbeTargets
{
    [MethodImpl(MethodImplOptions.NoInlining | MethodImplOptions.NoOptimization)]
    public static int DynamicProbeTarget(int value)
    {
        return (value * 2) + 1;
    }
}

public sealed class ValidationProfileException(string message) : Exception(message);
