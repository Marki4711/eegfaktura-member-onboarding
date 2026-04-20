import NextAuth from "next-auth";
import { authOptions } from "@/lib/auth";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const handler = NextAuth(authOptions) as any;

export async function GET(req: Request, ctx: unknown) {
  const url = new URL(req.url);
  const ts = new Date().toISOString();
  console.log(`[nextauth] ${ts} GET ${url.pathname}${url.search.slice(0, 80)}`);
  try {
    const res = await handler(req, ctx);
    console.log(`[nextauth] ${ts} GET → ${res?.status} location: ${res?.headers?.get("location") ?? "-"}`);
    return res;
  } catch (e) {
    console.error(`[nextauth] ${ts} GET ERROR:`, e);
    throw e;
  }
}

export async function POST(req: Request, ctx: unknown) {
  const url = new URL(req.url);
  const ts = new Date().toISOString();
  console.log(`[nextauth] ${ts} POST ${url.pathname}`);
  try {
    const res = await handler(req, ctx);
    console.log(`[nextauth] ${ts} POST → ${res?.status}`);
    return res;
  } catch (e) {
    console.error(`[nextauth] ${ts} POST ERROR:`, e);
    throw e;
  }
}
