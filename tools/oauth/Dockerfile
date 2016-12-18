FROM node:6

# only package.json
ADD package.json /entitlement/
RUN cd /entitlement; npm install

ADD . /entitlement

# See index.js
CMD ["/bin/bash", "-c", "node /entitlement/index.js >> /logs/${COMPONENT_NAME}_logs.log 2>&1"]
EXPOSE 8080