import { logger } from "../../../lib/logger";

const checkoutUrl = process.env.CHECKOUT_URL ?? "http://checkout:8080";
const checkoutUsername = process.env.CHECKOUT_USERNAME ?? "frontend.astronomy-store";
const checkoutPassword = process.env.CHECKOUT_PASSWORD ?? "changeme";
const checkoutAuthorization = `Basic ${Buffer.from(`${checkoutUsername}:${checkoutPassword}`).toString("base64")}`;

export async function POST(request) {
  const body = await request.text();

  const response = await fetch(`${checkoutUrl}/checkout/place-order`, {
    method: "POST",
    headers: {
      authorization: checkoutAuthorization,
      "content-type": "application/json",
    },
    body,
    cache: "no-store",
  });

  logger.info(
    { "http.route": "/api/checkout", "http.response.status_code": response.status },
    "POST /api/checkout processed",
  );

  return new Response(response.body, {
    status: response.status,
    headers: {
      "content-type":
        response.headers.get("content-type") ?? "application/json; charset=utf-8",
    },
  });
}
