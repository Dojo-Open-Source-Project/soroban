FROM    node:20-alpine3.18

ENV     NODE_ENV    production
ENV     APP_DIR     /home/node/app

RUN     set -ex && \
        apk --no-cache add gcc g++ make python3 curl cmake

# Update NPM to latest version
RUN     set -ex && \
        npm i -g npm

# Create app directory
RUN     mkdir "$APP_DIR"

# Copy app source files into APP_DIR
COPY    --chown=node:node   . "$APP_DIR"

# Install node modules required by the app
RUN     cd "$APP_DIR" && \
        npm install --omit=dev

WORKDIR "$APP_DIR"

ENTRYPOINT ["node", "soroban-client.js"]
