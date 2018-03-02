FROM node:9-alpine

RUN apk --update add --virtual git

RUN mkdir -p /var/agent
WORKDIR /var/agent

COPY . /var/agent
RUN npm install

CMD ["npm", "start"]