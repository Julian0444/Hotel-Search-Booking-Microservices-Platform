// Ejecutado por mongo-init una vez por arranque de Compose; idempotente.
const uri = 'mongodb://root:' + encodeURIComponent(process.env.MONGO_PASSWORD) + '@mongo:27017/admin?directConnection=true';
const admin = new Mongo(uri).getDB('admin');
try {
    const status = admin.runCommand({replSetGetStatus: 1});
    if (status.ok !== 1) {
        if (status.code !== 94) throw new Error(JSON.stringify(status));
        const init = admin.runCommand({replSetInitiate: {_id: 'rs0', members: [{_id: 0, host: 'mongo:27017'}]}});
        if (init.ok !== 1) throw new Error(JSON.stringify(init));
    }
} catch (error) {
    if (error.code !== 94) throw error;
    const init = admin.runCommand({replSetInitiate: {_id: 'rs0', members: [{_id: 0, host: 'mongo:27017'}]}});
    if (init.ok !== 1) throw new Error(JSON.stringify(init));
}
for (let attempt = 0; attempt < 60; attempt++) {
    if (admin.runCommand({hello: 1}).isWritablePrimary) {
        print('rs0 writable primary ready');
        quit(0);
    }
    sleep(1000);
}
throw new Error('rs0 did not elect a primary within 60 seconds');
