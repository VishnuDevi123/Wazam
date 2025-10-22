SEEK-TUNE


⸻

OVERVIEW

Seek-Tune is an AI-powered music discovery platform that combines a reverse-engineered audio fingerprinting engine (Shazam-style), a Go backend for matching and file management, a React frontend for user interaction, and a Mastra Agent for conversational, tool-enabled recommendations.
The project is containerized with Docker Compose so that the entire stack (frontend, backend, and agent) can be built and run consistently in one command.

⸻

KEY COMPONENTS AND CONCEPTS

AUDIO FINGERPRINTING (SHAZAM-LIKE)

The goal of the fingerprinting system is to identify a song from a short audio snippet, whether it is recorded audio, device playback, or even a user hum.

The process begins when the user records a short audio segment of about twenty seconds. This audio is then preprocessed in the frontend using FFmpeg to ensure it is in mono format and resampled to a fixed rate of 44.1 kHz.

The core of the fingerprinting is implemented as a WebAssembly (WASM) module that runs directly in the browser. This module performs digital signal processing to extract spectral peaks across time, encoding them as address and anchorTime pairs that represent high-energy frequency events. The resulting fingerprint map (address → anchorTime) emulates the constellation map method used by Shazam-like systems.

Once generated, the fingerprint map is sent to the Go backend, which performs a hash-based lookup across a precomputed database of song fingerprints. Matches are scored based on time alignment and the number of consistent peaks. The top-ranked matches are returned with metadata such as the song title, artist, and album.

This approach is fast, noise-tolerant, and memory-efficient, since it stores hash pairs instead of entire waveform data.

⸻

BACKEND (GO)

The backend is responsible for handling all song metadata, fingerprint storage, and matching logic. It receives incoming fingerprint queries and returns ranked matches.

It also provides additional endpoints for tasks such as downloading songs, generating WAV files, and managing stored media.

Internally, it uses an SQLite database and a dedicated songs directory to store references and fingerprint files.
It supports WebSocket communication for real-time interactions, such as live download status and match results.

The production Docker image also bundles ffmpeg and yt-dlp to support media processing and conversion directly within the container.

⸻

FRONTEND (REACT)

The React frontend serves as the primary user interface for Seek-Tune. It allows users to record or capture audio directly from their device, convert it to mono WAV format using FFmpeg (executed in-browser), and perform WASM-based fingerprint generation before sending any data to the server.

Because fingerprinting happens entirely in-browser, user privacy is preserved as no raw audio leaves the client.

Once a match is found, the frontend displays the results through a clean carousel interface, allowing users to play clips, preview matches, and download files.

A MastraChat component is also included in the UI, which connects to the Mastra Agent backend, allowing users to interact with the system conversationally, ask for similar songs, and explore recommendations.
Socket.IO is used for real-time updates from the Go backend to keep the interface dynamic and responsive.

⸻

MASTRA AGENT (CONVERSATIONAL AI)

The Mastra Agent enables natural-language conversation and intelligent recommendations. It runs as a separate container (mastra-agent) and exposes an API endpoint that the frontend communicates with.

It uses preconfigured tools such as songRecommenderTool to fetch Spotify-like song recommendations and return contextual, human-readable responses.

The agent supports follow-up questions, such as “find me more songs like this” or “recommend me similar artists,” making the experience interactive and personalized.

⸻

ARCHITECTURE DIAGRAM

[User Browser]
    ├── React Frontend (WASM Fingerprinting + UI)
    ├── Socket.IO → Go Backend (Matching, Downloads, Status)
    └── HTTP → Mastra Agent (Conversational Tools & Recommendations)

[Go Backend] ↔ [Fingerprint DB / Song Storage]
[Mastra Agent] ↔ [Tools (Spotify API, SongRecommenderTool, etc.)]


⸻

HOW IT WORKS

When a user records a short clip, the frontend captures the audio and converts it to mono WAV format using FFmpeg.
The WebAssembly fingerprinting function runs locally in the browser and generates a fingerprint map from the audio.
This map is sent to the Go backend, which performs a lookup and computes similarity scores.
The backend returns the best matches, and the frontend displays them to the user.
If the user requests further recommendations, the chat interface sends the query to the Mastra Agent, which fetches related suggestions using its tools and responds conversationally.

⸻

DOCKER AND DEPLOYMENT

The entire Seek-Tune stack is containerized using Docker Compose.
The Mastra Agent runs in its own container, while the Go backend container also serves the compiled React frontend.

To build and start the entire system from the repository root, use:

docker compose up --build

To start without rebuilding:

docker compose up

To stop and remove all containers:

docker compose down

The Docker network allows the frontend to communicate with the Mastra Agent at http://mastra-agent:4111 and the Go backend through its own internal address.
If any API endpoints are hard-coded for local testing, replace them with environment variables before building for production.

⸻

DEVELOPMENT NOTES

The frontend build uses the environment variable REACT_APP_BACKEND_URL to connect to the backend during runtime.
The fingerprinting WebAssembly file (main.wasm) must be accessible under /main.wasm in the production build, and the function generateFingerprint() should be globally exposed to the browser.

The Mastra Agent starts with the command npm run dev in the mastra-agent directory.
If issues occur during build (for example, missing native dependencies), remove the package-lock.json file and reinstall dependencies using npm ci.

The songRecommenderTool currently uses a basic search-based recommendation method. You may replace it with an OAuth-secured Spotify API integration for richer metadata and personalized responses.

⸻

FUTURE FEATURES AND EXTENSIONS

Planned enhancements include a hum-to-search feature that maps melodic contours to fingerprints, an emotion-based mood tagging model to recommend songs by sentiment, and collaborative filtering for playlist generation.
A lightweight mobile SDK is also planned to support native iOS and Android capture for portable, offline fingerprinting.

⸻

TROUBLESHOOTING

Working with legacy code was one of the most significant challenges in this project.
Many outdated libraries required replacement or modification to support modern dependency chains.

If the Mastra Agent fails to start, ensure that npm run dev works locally and that the containerized node modules are compatible with your system architecture.

If fingerprint mismatches occur, confirm that the sample rate and mono conversion are consistent and that the WebAssembly file is being properly loaded in the frontend environment.

⸻

ADDITIONAL INFORMATION

Before running the Mastra Agent, create an .env file in the Mastra workspace and include your OpenAI API key.
You will also need to include your Spotify Client ID and Spotify Client Secret in an .env file inside the server folder.

⸻

REFERENCES

Huge thanks to Chigozirim for providing the original legacy code that this project was built upon.
Reference video: https://www.youtube.com/watch?v=a0CVCcb0RJM

For nerds:
https://drive.google.com/file/d/1ahyCTXBAZiuni6RTzHzLoOwwfTRFaU-C/view

⸻


![alt text](image-2.png)
![alt text](image-3.png)