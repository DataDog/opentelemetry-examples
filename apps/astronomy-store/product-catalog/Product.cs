public sealed record Product(
    string Id,
    string Name,
    string Description,
    string Picture,
    Price Price,
    string[] Categories);

public sealed record Price(
    string CurrencyCode,
    long Units,
    int Nanos);
