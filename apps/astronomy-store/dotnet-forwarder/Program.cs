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
        var result = consumer.Consume(cts.Token);
        logger.LogInformation("forward_order key={Key}", result.Message.Key);
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

static string Env(string key, string defaultValue) =>
    Environment.GetEnvironmentVariable(key) is { Length: > 0 } value ? value : defaultValue;
