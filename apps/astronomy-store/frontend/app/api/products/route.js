const productCatalogUrl =
  process.env.PRODUCT_CATALOG_URL ?? "http://product-catalog:8080";

export async function GET() {
  const response = await fetch(`${productCatalogUrl}/products`, {
    cache: "no-store",
  });

  return new Response(response.body, {
    status: response.status,
    headers: {
      "content-type":
        response.headers.get("content-type") ?? "application/json; charset=utf-8",
    },
  });
}
