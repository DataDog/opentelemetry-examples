var builder = WebApplication.CreateBuilder(args);
builder.Logging.AddJsonConsole(options => options.IncludeScopes = true);
var app = builder.Build();

app.MapGet("/products", () => Results.Ok(Catalog.Products));
app.MapGet("/product/{uid}", Program.GetProduct);
app.MapGet("/health", () => Results.Ok());

app.Run();

// Explicit declarations keep Program and GetProduct discoverable by Live Debugger's symbol extractor.
partial class Program
{
    public static IResult GetProduct(string uid, ILogger<Program> logger)
    {
        logger.LogInformation("GetProduct invoked with uid={Uid}", uid);
        var product = Catalog.Products.FirstOrDefault(product => product.Id == uid);
        return product is null ? Results.NotFound() : Results.Ok(product);
    }
}
