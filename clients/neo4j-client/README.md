# Neo4j Client
The Neo4j client collects AccessLogs from SentryFlow, stores them, and visualizes them.

## Neo4j Client Deployment
Neo4j client can be deployed using kubectl command. The deployment can be accomplished with the following
commands:
```bash
$ cd SentryFlow/deployments
$ kubectl apply -f neo4j-client.yaml
```

## Neo4j settings
### Step 1. Create Neo4j account
Go to https://neo4j.com/ and create an account

### Step 2. Create Neo4j Instance
Remember the Username and Password you created when creating the instance.

### Step 3. Modify env value in neo4j-client.yaml file.
Put the Connection URI specified in the instance into NEO4J_URI, and the information created in Step 2 into NEO4J_USERNAME and NEO4J_PASSWORD, respectively.

```bash
env:
- name: NEO4J_URI
  value: ""
- name: NEO4J_USERNAME
  value: ""
- name: NEO4J_PASSWORD
  value: ""
```

## Neo4j client options
These are the default env value in the neo4j-client.yaml file.
```bash
env:
- name: NODE_LEVEL
  value: "simple"
- name: EDGE_LEVEL
  value: "simple"
```

If you want to change the default env value, you can refer to the following options.
```bash
env:
- name: NODE_LEVEL
  value: {"simple"|"detail"}
- name: EDGE_LEVEL
  value: {"simple"|"detail"}
```

## Example with robot-shop
### Example 1 (NODE_LEVEL: simple, EDGE_LEVEL: simple)
![Neo4j example1](/docs/neo4j_01.png)

### Example 2 (NODE_LEVEL: simple, EDGE_LEVEL: detail)
![Neo4j example2](/docs/neo4j_02.png)

### Example 3 (NODE_LEVEL: detail, EDGE_LEVEL: detail)
![Neo4j example3](/docs/neo4j_03.png)
