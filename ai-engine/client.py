import os
import uuid

import grpc

from protobuf import sentryflow_metrics_pb2_grpc
from protobuf import sentryflow_metrics_pb2

if __name__ == "__main__":
    try:
        listen_addr = os.environ["AI_ENGINE_ADDRESS"]
    except KeyError:
        listen_addr = "0.0.0.0:5000"

    with grpc.insecure_channel(listen_addr) as channel:
        stub = sentryflow_metrics_pb2_grpc.SentryFlowMetricsStub(channel)
        req = sentryflow_metrics_pb2.APIClassificationRequest(paths=["/api/test", "/api/test/" + str(uuid.uuid4())])

        try:
            response_stream = stub.GetAPIClassification(req)
            for response in response_stream:
                print("Response: ", str(response))
        except grpc.RpcError as e:
            print("Error occurred during RPC:", e)