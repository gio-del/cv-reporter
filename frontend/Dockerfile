FROM node:24-alpine
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm install
COPY . .
EXPOSE 5173
# node_modules is bind-mounted over by docker-compose's anonymous volume, so
# it can go stale against package.json after a pull. Reconcile it on every
# start rather than only at image build time.
CMD ["sh", "-c", "npm install && npm run dev"]
