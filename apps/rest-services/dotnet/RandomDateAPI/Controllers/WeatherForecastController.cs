using System;
using Microsoft.AspNetCore.Mvc;

namespace RandomDateAPI.Controllers
{
    [ApiController]
    [Route("[controller]")]
    public class RandomDateController : ControllerBase
    {
        [HttpGet]
        public DateTime Get()
        {
            return GenerateRandomDate();
        }

        private static DateTime GenerateRandomDate()
        {
            Random random = new Random();
            int year = random.Next(1900, 2100);
            int month = random.Next(1, 13);
            int day = random.Next(1, DateTime.DaysInMonth(year, month) + 1);

            return new DateTime(year, month, day);
        }
    }
}