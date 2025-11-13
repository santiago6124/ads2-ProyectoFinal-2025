db = db.getSiblingDB('cryptosim_orders');
db.createCollection('orders');
db.orders.createIndex({ "user_id": 1, "created_at": -1 });
db.orders.createIndex({ "status": 1, "created_at": -1 });