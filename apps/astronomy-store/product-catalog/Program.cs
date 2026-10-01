var builder = WebApplication.CreateBuilder(args);
// One JSON object per log line on stdout. IncludeScopes adds ASP.NET Core's per-request scope,
// which carries the TraceId and SpanId of the current request for logs/traces correlation.
builder.Logging.AddJsonConsole(options => options.IncludeScopes = true);
var app = builder.Build();

app.MapGet("/products", () => Results.Ok(Catalog.Products));
app.MapGet("/product/{uid}", GetProduct);
app.MapGet("/health", () => Results.Ok());

app.Run();

static IResult GetProduct(string uid, ILogger<Program> logger)
{
    logger.LogInformation("GetProduct invoked with uid={Uid}", uid);
    var product = Catalog.Products.FirstOrDefault(product => product.Id == uid);
    return product is null ? Results.NotFound() : Results.Ok(product);
}
