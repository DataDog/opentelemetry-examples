using System.Diagnostics;
using System.Diagnostics.Metrics;
using OpenTelemetry.Exporter;
using OpenTelemetry.Logs;
using OpenTelemetry.Metrics;
using OpenTelemetry.Resources;
using OpenTelemetry.Trace;
using RandomDateAPI;

var builder = WebApplication.CreateBuilder(args);

// Add services to the container.

builder.Services.AddControllers();
// Learn more about configuring Swagger/OpenAPI at https://aka.ms/aspnetcore/swashbuckle
builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen();

builder.Services.AddSingleton<INewshoreLogger, NewshoreLogger>();
builder.Services.AddSingleton(new Meter(DiagnosticsConfig.MeterName));


builder.Logging.ClearProviders();
// OpenTelemetry Setup
builder.Services.AddOpenTelemetry()
    .WithTracing(tracerProviderBuilder =>
        tracerProviderBuilder
            .AddSource(DiagnosticsConfig.ActivitySource.Name)
            .ConfigureResource(resource => resource
                .AddService(DiagnosticsConfig.ServiceName))
            .AddAspNetCoreInstrumentation()
            .AddOtlpExporter(opt => opt.Endpoint = new Uri("http://localhost:4317")))
    .WithMetrics(metricsProviderBuilder =>
        metricsProviderBuilder
            .ConfigureResource(resource => resource
                .AddService(DiagnosticsConfig.ServiceName))
            .AddAspNetCoreInstrumentation()
            .AddRuntimeInstrumentation()
            .AddMeter(DiagnosticsConfig.MeterName)
            .AddOtlpExporter(opt => opt.Endpoint = new Uri("http://localhost:4317")));

// builder.Logging.AddLog4net()
//builder.Logging.AddJsonConsole();
builder.Logging.AddOpenTelemetry(options =>
{
    // Todo #1 Andrei: Validate kung yung time format nagbabago pag pinasa sa datadog agent natin
    // Todo Andrei #2: Gumamit ng ibang logger pero ginagamit si ILogger. gamit ILoggerFactory
    // Estimate 1-2h
    options.IncludeScopes = true;
    options.IncludeFormattedMessage = true;
    options.ParseStateValues = true;
    options
        .SetResourceBuilder(
            ResourceBuilder.CreateDefault()
                .AddService(DiagnosticsConfig.ServiceName))
        .AddConsoleExporter(exporterOptions => exporterOptions.Targets = ConsoleExporterOutputTargets.Console)
        .AddOtlpExporter(opt => { opt.Endpoint = new Uri("http://localhost:4317"); });
});

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

public static class DiagnosticsConfig
{
    public const string ServiceName = "random-date-dotnet-otel";
    public const string MeterName = "random-date-dotnet-otel.metric";
    public static ActivitySource ActivitySource = new(ServiceName);
}