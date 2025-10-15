# Mastra Agent Dockerfile (fully fixed)
FROM node:20-bullseye-slim

WORKDIR /app

# Copy only package files first to leverage caching
COPY package*.json ./

# Clean NPM cache and install fresh dependencies (no prebuilt rollup binaries)
RUN npm cache clean --force && \
    npm install --ignore-scripts && \
    npm rebuild rollup --verbose

# Copy rest of the source
COPY . .

EXPOSE 4111

CMD ["npm", "run", "dev"]