OVERVIEW
![alt text](image-1.png)
Seek-Tune is an AI-powered music discovery platform that combines a reverse-engineered audio fingerprinting engine (Shazam-style), a Go backend for matching and file management, a React frontend for user interaction, and a Mastra Agent for conversational, tool-enabled recommendations. The project is containerized with Docker Compose so the entire stack (frontend/backend/agent) can be built and run consistently.


KEY COMPONENTS & CONCEPTS

1. AUDIO FINGERPRINTING (Shazam-LIKE)
GOAL: Identify a song from a short audio snippet (hum, recorded audio, or system audio capture) quickly and reliably.

    PROCESS:
	•	Capture: Short audio capture (up to ~20 seconds) from microphone or device.
	•	Preprocessing: Convert to mono, resample to a fixed sample rate (e.g., 44.1 kHz), remove extra channels — handled in the frontend with FFmpeg, and validated before fingerprinting.
	WASM-BASED FINGERPRINTING:
	•	A WebAssembly (WASM) module (loaded by the frontend) implements the core fingerprinting algorithm so heavy DSP runs in the browser.
	•	The WASM module extracts spectral peaks across time and encodes address and anchorTime pairs representing high-energy time-frequency events. This emulates the constellation map approach used by Shazam-style algorithms.
	•	The frontend converts the WASM output into a compact fingerprint map (address → anchorTime) and sends it to the server.
	Server Matching (Go backend):
	•	The backend receives the fingerprint map and performs a hashed lookup against an internal index.
	•	Matches are scored using anchor-time alignment and vote-counting (how many addresses align with consistent time offsets).
	•	Top matches are returned with metadata (title, artist, album, links).
	•	WHY THIS APPROACH: Fast lookups, robust to noise and small tempo/pitch variations, and efficient storage by hashing peak pairs rather than entire audio.

2. BACKEND (Go)
    ROLES:
	•	Receive and store song metadata and fingerprints.
	•	Accept incoming fingerprint queries and return ranked matches.
	•	Provide download/processing endpoints (for saving audio, generating WAV files, handling song assets).
	NOTABLES:
	•	Uses an internal songs directory and DB (configurable—default SQLite).
	•	Exposes WebSocket endpoints for real-time interactions (download/processing status, match streaming).
	•	Bundled with necessary native tools (ffmpeg, yt-dlp) in the production Docker image for conversion/download tasks.

3. FRONTEND (React)
	FEATURES:
	•	Record or capture audio (device-screen, microphone).
	•	Convert to mono WAV using FFmpeg (in-browser).
	•	Use the WASM fingerprint function to produce fingerprint data without sending raw audio to the server (privacy + speed).
	•	Show carousel of matches, play clips, and provide download options.
	•	Integrated chat widget that lets users interact with the Mastra Agent (ask for similar songs, context, or follow-up questions).
	•	ARCHITECTURE NOTES:
	•	The frontend includes a MastraChat component that posts user messages to the Mastra agent API endpoint and renders conversational output.
	•	Uses socket.io for real-time updates from the Go backend.

4. MASTRA AGENT (Conversational AI)
	•	ROLE: Provide natural-language interaction, call tools (e.g., songRecommenderTool) to fetch Spotify-like recommendations, and return structured and human-friendly responses.
	•	INTEGRATION:
	•	The frontend routes chat requests to the Mastra Agent API (running in its own container).
	•	The agent is configured with tools for fetching songs and can call them as functions; the tool outputs are surfaced in the chat UI.
	•	CAPABILITIES: Recommend similar songs, list reasons (genre/mood/tempo), and accept follow-up clarifying queries.

ARCHITECTURE DIAGRAM (TEXTUAL)

[User Browser]
  ├-> React Frontend (WASM fingerprinting + UI)
  ├─> Socket.IO => Go Backend (matching, downloads, status)
  └─> HTTP => Mastra Agent (conversational queries & tools)
[Go Backend] <=> [Fingerprint DB / Songs storage]
[Mastra Agent] <=> [Tools (songRecommenderTool, Spotify APIs, etc.)]


HOW IT WORKS — A TYPICAL FLOW
	1.	User clicks Record in the frontend.
	2.	Frontend captures audio, converts to mono WAV using FFmpeg in-browser.
	3.	The WASM fingerprint function runs in-browser and returns a fingerprint map.
	4.	The frontend POSTs the fingerprint to the Go backend via socket or HTTP.
	5.	Backend performs matching, scores results, returns best matches to frontend; frontend displays them.
	6.	If the user asks follow-up questions, the frontend sends chat messages to the Mastra Agent API; the agent uses tools (including the songRecommenderTool) to fetch richer recommendations and returns conversational responses.


DOCKER & DEPLOYMENT (RUNNING THE STACK WITH DOCKER)

The entire application is containerized. The Mastra Agent runs in its own container and the Seek-Tune backend runs in another. The frontend build is included in the Go server container.

Build & Start (recommended)

# From the repository root
docker compose up --build

Start (without rebuilding)

docker compose up

Stop and Remove Containers

docker compose down

Notes
	•	The project uses Docker Compose networking so the frontend can reach the Mastra agent at the agent container host (e.g., http://mastra-agent:4111 from other containers) and the Go backend at its own service address.
	•	If you hard-coded local endpoints in the frontend during development, update them to use environment variables before building the image.



DEVELOPMENT NOTES
	•	Environment Variables: Use REACT_APP_BACKEND_URL during the React build to direct frontend API calls to the correct backend address.
	•	WASM Fingerprinter: The fingerprinting WASM must be available at /main.wasm in the frontend build; ensure the go wasm runner is included and generateFingerprint is exposed to window.
	•	Mastra Agent Start: The Mastra agent should be started via npm run dev within the mastra-agent directory (containerized in docker compose). Ensure node_modules are present (or install them inside the container during build). If installing in-container fails due to platform-specific optional native modules, remove lockfiles / re-run npm ci or use build args to match your target architecture.
	•	Spotify/Third-Party APIs: songRecommenderTool uses a simple search-based approach (no OAuth) for demo; replace or extend with OAuth-backed producer APIs if you need richer metadata.


FUTURE FEATURES & EXTENSIONS
	•	Hum-to-Search: Add a model-based hum recognition pipeline for mapping melodic contours to existing fingerprints (requires additional training/data).
	•	Emotion / Mood Tagging: Use ML to tag songs by mood and recommend by user mood signals.
	•	Playlists & Collaborative Filtering: Add user accounts and build personalized playlists derived from agent conversations and match history.
	•	Mobile SDK: Expose a compact fingerprinting SDK for iOS/Android to capture audio without browser restrictions.


TROUBLESHOOTING
	•	Working on legacy code was one of the major issues I have faced. There were many libraries which were outdated and needed to be replaced with new and advanced ones
	•	Mastra Agent Not Starting: Confirm npm run dev works locally inside mastra-agent; if containerized, ensure node modules are installed for the container architecture and optional native libs are handled.
	•	Fingerprint Mismatch: Verify audio sample rate / mono conversion steps match the expected values before fingerprinting. Also confirm WASM is loaded successfully and generateFingerprint returns a valid structure.


Additionaly important information!
-->Make sure to create an env file and include your OpenAI api key in the Mastra workspace.

-->Additionaly make sure to have spotify clinet ID and secret phrase and include them in the env file in the 'server' folder


References:
--->Huge shortout to "Chigozirim" for his legacy code to work on
--->https://www.youtube.com/watch?v=a0CVCcb0RJM

For nerds:
https://drive.google.com/file/d/1ahyCTXBAZiuni6RTzHzLoOwwfTRFaU-C/view

![alt text](image-1.png)