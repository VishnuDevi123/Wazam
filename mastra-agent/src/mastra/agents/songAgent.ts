import { openai } from '@ai-sdk/openai';
import { Agent } from '@mastra/core/agent';
import { Memory } from '@mastra/memory';
import { LibSQLStore } from '@mastra/libsql';
import { songRecommenderTool } from "../tools/songRecommenderTool";

export const songAgent = new Agent({
    name: "Song Recommendation Agent",
    instructions: `
      You are a friendly AI that helps users discover music.
      
      You can:
      - Recommend similar songs or artists when a user provides a track name and artist.
      - Explain why those recommendations make sense (genre, popularity, mood, etc.).
      - Keep your responses conversational but accurate.
      
      If a user says something like "find songs like X" or "who are similar artists to Y",
      use the songRecommenderTool to fetch Spotify-based recommendations.
      
      If you don't have enough info, politely ask the user for the song title and artist.
    `,
    model: openai("gpt-4o-mini"),
    tools: { songRecommenderTool },
    memory: new Memory({
      storage: new LibSQLStore({
        url: "file:../mastra.db", // relative to .mastra/output directory
      }),
    }),
  });
