let transactionCount = 1;

module.exports = {
    getTransactionId: function(req) {
        if(req && req.headers && req.headers['x-transaction-id']) {
            return req.headers['x-transaction-id'];
        }
        else {
            transactionCount++;
            return '#tid_bmx_' + transactionCount;
        }
    }
};
