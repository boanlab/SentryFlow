# SPDX-License-Identifier: Apache-2.0

"""SentryFlow AI API Classification Engine"""

from concurrent import futures
from collections import Counter

import os
import grpc

from stringlifier.api import Stringlifier
from protobuf import sentryflow_metrics_pb2_grpc
from protobuf import sentryflow_metrics_pb2


class HandlerServer:
    """
    Class for gRPC Servers
    """
    def __init__(self):
        try:
            self.listen_addr = os.environ["AI_ENGINE_ADDRESS"]
        except KeyError:
            self.listen_addr = "0.0.0.0:5000"

        self.server = None
        self.grpc_servers = []

    def init_grpc_servers(self):
        """
        init_grpc_servers method that initializes and registers gRPC servers
        :return: None
        """
        self.server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
        self.grpc_servers.append(APIClassificationServer())  # @todo: make this configurable

        grpc_server: GRPCServer
        for grpc_server in self.grpc_servers:
            grpc_server.register(self.server)

    def serve(self):
        """
        serve method that starts serving gRPC servers, this is blocking function.
        :return: None
        """
        self.server.add_insecure_port(self.listen_addr)

        print(f"[INFO] Starting to serve on {self.listen_addr}")
        self.server.start()
        self.server.wait_for_termination()


class GRPCServer:
    """
    Abstract class for an individual gRPC Server
    """
    def register(self, server):
        """
        register method that registers gRPC service to target server
        :param server: The server
        :return: None
        """

    def unregister(self, server):
        """
        unregister method that unregisters gRPC service from target server
        :param server: The server
        :return: None
        """


class APIClassificationServer(sentryflow_metrics_pb2_grpc.APIClassificationServicer, GRPCServer):
    """
    Class for API Classification Server using Stringlifier
    """

    def __init__(self):
        self.stringlifier = Stringlifier()
        print("[Init] Successfully initialized APIClassificationServer")

    def register(self, server):
        sentryflow_metrics_pb2_grpc.add_APIClassificationServicer_to_server(self, server)

    def ClassifyAPIs(self, request_iterator, context):
        """
        GetAPIClassification method that runs multiple API ML Classification at once
        :param request_iterator: The requests
        :param context: The context
        :return: The results
        """

        for req in request_iterator:
            all_paths = req.API
            # for paths in all_paths:
            ml_results = self.stringlifier(all_paths)

            ml_counts = Counter(ml_results)
            print(f"{all_paths} -> {ml_counts}")

            yield sentryflow_metrics_pb2.APIClassificationResponse(APIs=ml_counts)


if __name__ == '__main__':
    hs = HandlerServer()
    hs.init_grpc_servers()
    hs.serve()
