using Confluent.Kafka;
using Microsoft.Extensions.Logging;

using var loggerFactory = LoggerFactory.Create(builder => builder.AddJsonConsole());
var logger = loggerFactory.CreateLogger("DotnetForwarder");

var broker = Env("KAFKA_ADDR", "kafka:9092");
var inputTopic = Env("KAFKA_TOPIC", "node-forwarder-orders");
var outputTopic = Env("KAFKA_OUTPUT_TOPIC", "dotnet-forwarder-orders");

using var cts = new CancellationTokenSource();
Console.CancelKeyPress += (_, e) => { e.Cancel = true; cts.Cancel(); };
AppDomain.CurrentDomain.ProcessExit += (_, _) => cts.Cancel();

var consumerConfig = new ConsumerConfig
{
    BootstrapServers = broker,
    GroupId = "dotnet-forwarder",
    AutoOffsetReset = AutoOffsetReset.Earliest,
};
var producerConfig = new ProducerConfig { BootstrapServers = broker };

using var consumer = new ConsumerBuilder<string, string>(consumerConfig).Build();
using var producer = new ProducerBuilder<string, string>(producerConfig).Build();
consumer.Subscribe(inputTopic);
logger.LogInformation("dotnet-forwarder started: {InputTopic} -> {OutputTopic}", inputTopic, outputTopic);

try
{
    while (!cts.IsCancellationRequested)
    {
        ConsumeResult<string, string> result;
        try
        {
            result = consumer.Consume(cts.Token);
        }
        catch (ConsumeException e) when (e.Error.Code == ErrorCode.UnknownTopicOrPart)
        {
            logger.LogWarning("waiting for topic {InputTopic} to exist", inputTopic);
            await Task.Delay(TimeSpan.FromSeconds(2), cts.Token);
            continue;
        }
        logger.LogInformation("forward_order key={Key}", result.Message.Key);
        await Task.Delay(ProcessingDelay(), cts.Token);
        await producer.ProduceAsync(outputTopic, new Message<string, string> { Key = result.Message.Key, Value = result.Message.Value }, cts.Token);
    }
}
catch (OperationCanceledException)
{
}
finally
{
    producer.Flush(TimeSpan.FromSeconds(10));
    consumer.Close();
}

static TimeSpan ProcessingDelay()
{
    var gaussian = Math.Sqrt(-2 * Math.Log(1 - Random.Shared.NextDouble())) * Math.Cos(2 * Math.PI * Random.Shared.NextDouble());
    return TimeSpan.FromMilliseconds(Math.Clamp(500 + 150 * gaussian, 0, 990));
}

static string Env(string key, string defaultValue) =>
    Environment.GetEnvironmentVariable(key) is { Length: > 0 } value ? value : defaultValue;
