FROM node:9-alpine

RUN mkdir -p /var/agent
WORKDIR /var/agent

COPY . /var/agent

RUN apk add --no-cache --virtual .build-deps git python make gcc g++ && \
    npm install && \
    npm cache clean --force && \
    apk del .build-deps

CMD ["npm", "start"]