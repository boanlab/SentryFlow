from pymongo import MongoClient
from stringlifier.api import Stringlifier
from flask import Flask

app = Flask(__name__)
s = Stringlifier()

@app.route('/api_metrics')
def api_metrics():
    # Connect to MongoDB
    client = MongoClient('mongodb://mongo:27017')
    # Access the numbat database
    db = client.numbat
    # Access the access-logs collection
    collection = db['access-logs']
    # Retrieve all documents from the collection
    logs = list(collection.find({}))
    # Close the MongoDB connection
    client.close()

    paths = list()
    # Print out all entries
    for log in logs:
        paths.append(log["path"])

    parsed = s(paths)
    print(set(parsed))

    return str(set(parsed))




if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000)