# Builder
FROM node:22-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY . .
RUN npm run build
# Ensure public directory exists even if empty
RUN mkdir -p /app/public

# Runner
FROM node:22-alpine
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm install --production --no-audit --no-fund
COPY --from=builder /app/.next .next
COPY --from=builder /app/public public
EXPOSE 3000
ENV NODE_ENV=production
CMD ["npm","run","start"]
