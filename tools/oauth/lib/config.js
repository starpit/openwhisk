module.exports = {
    servicePort: 8080,

    edgeHost: process.env.EDGE_HOST,
    routerHost: process.env.ROUTER_HOST,

    consulHost: process.env.CONSUL_HOST,
    consulPort: process.env.CONSUL_PORT,

    statsdHost: process.env.STATSD_HOST,
    statsdPort: process.env.STATSD_PORT,

    cfClientId: process.env.CF_CLIENT_ID,
    cfClientSecret: process.env.CF_CLIENT_SECRET,
    cfUaaEndpoint: process.env.CF_UAA_ENDPOINT,
    cfApiEndpoint: process.env.CF_API_ENDPOINT,

    spClientId: process.env.SP_CLIENT_ID,
    spClientSecret: process.env.SP_CLIENT_SECRET,
    spApiEndpoint: process.env.SP_API_ENDPOINT,

    amApiEndpoint: process.env.AM_API_ENDPOINT,

    continuousRefresh: process.env.CONTINUOUS_REFRESH === 'true',

    dbUsername: process.env.DB_USERNAME || process.env.OW_DB_USERNAME,
    dbPassword: process.env.DB_PASSWORD || process.env.OW_DB_PASSWORD,
    dbProtocol: process.env.DB_PROTOCOL || process.env.OW_DB_PROTOCOL,
    dbHost: process.env.DB_HOST || process.env.OW_DB_HOST,
    dbAuths: process.env.DB_AUTHS
};
