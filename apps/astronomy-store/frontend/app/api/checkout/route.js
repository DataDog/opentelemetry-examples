const checkoutUrl = process.env.CHECKOUT_URL ?? "http://checkout:8080";

export async function POST(request) {
  const body = await request.text();

  const response = await fetch(`${checkoutUrl}/checkout/place-order`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body,
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
