const adUrl = process.env.AD_URL ?? "http://ad:8080";

export async function GET(request) {
  const { searchParams } = new URL(request.url);
  const contextKeys = searchParams.getAll("context_keys");

  const target = new URL("/ad/get-ads", adUrl);
  for (const contextKey of contextKeys) {
    target.searchParams.append("context_keys", contextKey);
  }

  const response = await fetch(target, {
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
