FROM node:20-bullseye-slim

WORKDIR /app

# Copy dependencies first
COPY package*.json ./

# Install local and global dependencies
RUN npm install -g mastra && \
    npm install --ignore-scripts && \
    npm rebuild rollup --verbose

# Copy source files
COPY . .

EXPOSE 4111

# Use your same local command
CMD ["npm", "run", "dev"]