import { createTool } from "@mastra/core/tools";
import { z } from "zod";
import dotenv from "dotenv";

dotenv.config();

/**
 * songRecommenderTool
 * Fetches similar artists/songs from Spotify’s public API.
 */
async function getSpotifyAccessToken(): Promise<string> {
  const clientId = process.env.SPOTIFY_CLIENT_ID 
  const clientSecret = process.env.SPOTIFY_CLIENT_SECRET;
  const authString = Buffer.from(`${clientId}:${clientSecret}`).toString("base64");

  const res = await fetch("https://accounts.spotify.com/api/token", {
    method: "POST",
    headers: {
      Authorization: `Basic ${authString}`,
      "Content-Type": "application/x-www-form-urlencoded",
    },
    body: "grant_type=client_credentials",
  });

  if (!res.ok) throw new Error("Failed to get Spotify access token");
  const data = await res.json();
  return data.access_token;
}

// Core tool definition
export const songRecommenderTool = createTool({
  id: "songRecommenderTool",
  description:
    "Fetches similar songs using Spotify’s search API (no OAuth required). Uses artist and track name to find matching songs and suggest other songs by the same artist or similar titles.",
  inputSchema: z.object({
    title: z.string().describe("The song title to search for"),
    artist: z.string().optional().describe("The artist name (optional, improves accuracy)"),
  }),
  outputSchema: z.object({
    recommendations: z
      .array(
        z.object({
          title: z.string(),
          artist: z.string(),
          album: z.string().optional(),
          spotifyUrl: z.string().optional(),
        })
      )
      .describe("List of recommended songs from Spotify"),
  }),

  execute: async ({ context }) => {
    const { title, artist } = context;
    const token = await getSpotifyAccessToken();

    // --- Step 1: search for the main song ---
    const query = artist
      ? encodeURIComponent(`${title} artist:${artist}`)
      : encodeURIComponent(title);

    const searchUrl = `https://api.spotify.com/v1/search?q=${query}&type=track&limit=5`;

    const res = await fetch(searchUrl, {
      headers: { Authorization: `Bearer ${token}` },
    });

    if (!res.ok) {
      throw new Error(`Spotify search failed with status ${res.status}`);
    }

    const data = await res.json();

    // Extract base artist for heuristic “similar songs” search
    const primaryArtist =
      data.tracks?.items?.[0]?.artists?.[0]?.name || artist || "";

    // --- Step 2: Fallback heuristic search for similar songs ---
    const altQuery = encodeURIComponent(`artist:${primaryArtist}`);
    const altUrl = `https://api.spotify.com/v1/search?q=${altQuery}&type=track&limit=5`;

    const altRes = await fetch(altUrl, {
      headers: { Authorization: `Bearer ${token}` },
    });

    const altData = await altRes.json();

    const recommendations =
      altData.tracks?.items?.map((item: any) => ({
        title: item.name,
        artist: item.artists?.[0]?.name,
        album: item.album?.name,
        spotifyUrl: item.external_urls?.spotify,
      })) ?? [];

    if (recommendations.length === 0) {
      throw new Error("No recommendations found.");
    }

    return { recommendations };
  },
});