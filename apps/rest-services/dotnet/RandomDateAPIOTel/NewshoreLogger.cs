namespace RandomDateAPI;

public class NewshoreLogger(ILoggerFactory factory) : INewshoreLogger
{
    public void Log<T>(string message, LogLevel level)
    {
        // Microsoft.Extensions.ILogger
        var loggerInstance = factory.CreateLogger<T>();
        loggerInstance.Log(level, message);
    }
}

public interface INewshoreLogger
{
    void Log<T>(string message, LogLevel level);
}