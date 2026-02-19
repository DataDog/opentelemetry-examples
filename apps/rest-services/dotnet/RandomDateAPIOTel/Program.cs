using System.Diagnostics;
using OpenTelemetry.Resources;
using OpenTelemetry.Trace;
using OpenTelemetry.Metrics;

var builder = WebApplication.CreateBuilder(args);

// Add services to the container.
builder.Services.AddControllers();
builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen();

// Derive service name from OTel semantic conventions:
// 1. OTEL_SERVICE_NAME env var
// 2. service.name in OTEL_RESOURCE_ATTRIBUTES env var
// 3. Fallback default
var serviceName = Environment.GetEnvironmentVariable("OTEL_SERVICE_NAME")
    ?? GetServiceNameFromResourceAttributes()
    ?? "random-date-dotnet-otel";

var activitySource = new ActivitySource(serviceName);

// OpenTelemetry Setup
builder.Services.AddOpenTelemetry()
    .ConfigureResource(resource => resource
        .AddService(serviceName))
    .WithTracing(tracerProviderBuilder =>
        tracerProviderBuilder
            .AddSource(activitySource.Name)
            .AddAspNetCoreInstrumentation()
            .AddOtlpExporter())
    .WithMetrics(metricsProviderBuilder =>
        metricsProviderBuilder
            .AddAspNetCoreInstrumentation()
            .AddRuntimeInstrumentation()
            .AddOtlpExporter());

var app = builder.Build();

// Configure the HTTP request pipeline.
if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI();
}

app.UseHttpsRedirection();

app.UseAuthorization();

app.MapControllers();

app.Run();

static string? GetServiceNameFromResourceAttributes()
{
    var resourceAttributes = Environment.GetEnvironmentVariable("OTEL_RESOURCE_ATTRIBUTES");
    if (string.IsNullOrEmpty(resourceAttributes)) return null;

    foreach (var attribute in resourceAttributes.Split(','))
    {
        var parts = attribute.Split('=', 2);
        if (parts.Length == 2 && parts[0].Trim() == "service.name")
            return parts[1].Trim();
    }
    return null;
}