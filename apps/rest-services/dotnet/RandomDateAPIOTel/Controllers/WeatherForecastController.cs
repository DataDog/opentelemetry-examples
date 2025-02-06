using Microsoft.AspNetCore.Mvc;

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
        _logger.LogCritical("Get random date from RandomDateController. [{time}]",
            DateTime.UtcNow.ToString("yyyy-MM-dd HH:mm:ss.fff"));
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