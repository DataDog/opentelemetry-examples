using System.Diagnostics;
using OpenTelemetry.Logs;
using OpenTelemetry.Metrics;
using OpenTelemetry.Resources;
using OpenTelemetry.Trace;

var builder = WebApplication.CreateBuilder(args);

// Add services to the container.

builder.Services.AddControllers();
// Learn more about configuring Swagger/OpenAPI at https://aka.ms/aspnetcore/swashbuckle
builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen();

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
            .AddOtlpExporter(opt => opt.Endpoint = new Uri("http://localhost:4317")))
    .WithLogging(providerBuilder => providerBuilder
        .ConfigureResource(resource => resource
            .AddService(DiagnosticsConfig.ServiceName))
        .AddConsoleExporter()
        .AddOtlpExporter(opt => opt.Endpoint = new Uri("http://localhost:4317")));

//Add support to logging with SERILOG
// builder.Host.UseSerilog((context, configuration) =>
//     configuration.ReadFrom.Configuration(context.Configuration));


var app = builder.Build();

// Configure the HTTP request pipeline.
if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI();
}


//Add support to logging request with SERILOG
// app.UseSerilogRequestLogging();

app.UseHttpsRedirection();

app.UseAuthorization();

app.MapControllers();

app.Run();

public static class DiagnosticsConfig
{
    public const string ServiceName = "random-date-dotnet-otel";
    public static ActivitySource ActivitySource = new(ServiceName);
}