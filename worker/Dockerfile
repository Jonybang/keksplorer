FROM node:9-alpine

RUN apk --update add --virtual git

RUN mkdir -p /var/worker
WORKDIR /var/worker

COPY . /var/worker
RUN npm install

CMD ["npm", "start"]