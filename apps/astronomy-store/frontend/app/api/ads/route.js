import { logger } from "../../../lib/logger";

const adUrl = process.env.AD_URL ?? "http://ad:8080";
const adUsername = process.env.AD_USERNAME ?? "frontend.astronomy-store";
const adPassword = process.env.AD_PASSWORD ?? "changeme";
const adAuthorization = `Basic ${Buffer.from(`${adUsername}:${adPassword}`).toString("base64")}`;

const demoUsers = [
  { id: "user-1001", email: "ada.lovelace@example.com" },
  { id: "user-1002", email: "carl.sagan@example.com" },
  { id: "user-1003", email: "vera.rubin@example.com" },
  { id: "user-1004", email: "edwin.hubble@example.com" },
];

function getCurrentUser() {
  return demoUsers[Math.floor(Math.random() * demoUsers.length)];
}

export async function GET(request) {
  const { searchParams } = new URL(request.url);
  const contextKeys = searchParams.getAll("context_keys");
  const user = getCurrentUser();

  const target = new URL("/ad/get-ads", adUrl);
  for (const contextKey of contextKeys) {
    target.searchParams.append("context_keys", contextKey);
  }
  target.searchParams.set("user_id", user.id);
  target.searchParams.set("user_email", user.email);

  const response = await fetch(target, {
    cache: "no-store",
    headers: { authorization: adAuthorization },
  });

  logger.info(
    { "http.route": "/api/ads", "http.response.status_code": response.status },
    "GET /api/ads processed",
  );

  return new Response(response.body, {
    status: response.status,
    headers: {
      "content-type":
        response.headers.get("content-type") ?? "application/json; charset=utf-8",
    },
  });
}
