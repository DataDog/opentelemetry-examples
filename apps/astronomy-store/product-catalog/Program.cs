var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();

app.MapGet("/products", () => Results.Ok(Catalog.Products));
app.MapGet("/product/{uid}", GetProduct);
app.MapGet("/health", () => Results.Ok());

app.Run();

static IResult GetProduct(string uid)
{
    var product = Catalog.Products.FirstOrDefault(product => product.Id == uid);
    return product is null ? Results.NotFound() : Results.Ok(product);
}
