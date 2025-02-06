using System.Diagnostics;
using Microsoft.AspNetCore.Mvc;
using Serilog;
using Serilog.Context;

namespace RandomDateAPI.Controllers;

[ApiController]
[Route("[controller]")]
public class RandomDateController : ControllerBase
{
    private static readonly int[] value = new[] { 1, 2, 3 };
    private readonly ILogger<RandomDateController> _logger;

    public RandomDateController(ILogger<RandomDateController> logger)
    {
        _logger = logger;
    }

    [HttpGet]
    public async Task<OkObjectResult> Get()
    {
        // .NET Diagnostics: create a manual span
        using (var activity = DiagnosticsConfig.ActivitySource.StartActivity("andrei-test"))
        {
            activity?.SetTag("foo", 1);
            activity?.SetTag("bar", "Hello, World!");
            activity?.SetTag("baz", value);

            var waitTime = Random.Shared.NextDouble(); // max 1 seconds
            await Task.Delay(TimeSpan.FromSeconds(waitTime));

            activity?.SetStatus(ActivityStatusCode.Ok);

            // .NET Diagnostics: update the metric
        }

        _logger.LogCritical("Get random date from RandomDateController. {trace-id}, {span-id}",
            Activity.Current.TraceId.ToString(), Activity.Current.SpanId.ToString());
        using (_logger.BeginScope(new Dictionary<string, object>
               {
                   ["dd.trace_id"] = Activity.Current.TraceId.ToString(),
                   ["dd.span_id"] = Activity.Current.SpanId.ToString()
               }))
        {
            _logger.LogCritical("Get random date from RandomDateController. {trace-id}, {span-id}",
                Activity.Current.TraceId.ToString(), Activity.Current.SpanId.ToString());
        }

        var stringTraceId = Activity.Current.TraceId.ToString();
        var stringSpanId = Activity.Current.SpanId.ToString();

        var ddTraceId = Convert.ToUInt64(stringTraceId.Substring(16), 16).ToString();
        var ddSpanId = Convert.ToUInt64(stringSpanId, 16).ToString();

        using (LogContext.PushProperty("dd.trace_id", ddTraceId))
        using (LogContext.PushProperty("dd.span_id", ddSpanId))
        {
            Log.Logger.Information("Example log line with trace correlation info");
        }

        return Ok(GenerateRandomDate());
    }

    private static DateTime GenerateRandomDate()
    {
        var random = new Random();
        var year = random.Next(1900, 2100);
        var month = random.Next(1, 13);
        var day = random.Next(1, DateTime.DaysInMonth(year, month) + 1);

        return new DateTime(year, month, day);
    }
}