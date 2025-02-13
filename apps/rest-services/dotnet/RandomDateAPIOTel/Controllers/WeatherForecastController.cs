using System.Diagnostics.Metrics;
using Microsoft.AspNetCore.Mvc;

namespace RandomDateAPI.Controllers;

[ApiController]
[Route("[controller]")]
public class RandomDateController(ILogger<RandomDateController> logger, INewshoreLogger newshoreLogger, Meter meter)
    : ControllerBase
{
    private readonly Counter<int> s_datesGenerated =
        meter.CreateCounter<int>("otel.random_date_generator.dates_generated");

    [HttpGet]
    public async Task<OkObjectResult> Get()
    {
        newshoreLogger.Log<RandomDateController>("[NewshoreLogger] RandomDateController", LogLevel.Critical);
        logger.LogCritical("Get random date from RandomDateController. [{time}]",
            DateTime.UtcNow.ToString("yyyy-MM-dd HH:mm:ss.fff"));
        return Ok(GenerateRandomDate());
    }

    private DateTime GenerateRandomDate()
    {
        var random = new Random();
        var year = random.Next(1900, 2100);
        var month = random.Next(1, 13);
        var day = random.Next(1, DateTime.DaysInMonth(year, month) + 1);
        s_datesGenerated.Add(1);
        return new DateTime(year, month, day);
    }
}