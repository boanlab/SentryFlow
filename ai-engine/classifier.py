# SPDX-License-Identifier: Apache-2.0

"""SentryFlow AI Engine for API Classification"""

from concurrent import futures
from collections import Counter

import os
import grpc

from protobuf import sentryflow_metrics_pb2
from protobuf import sentryflow_metrics_pb2_grpc

from stringlifier.api import Stringlifier


class HandlerServer:
    """
    Class for gRPC Servers
    """
    def __init__(self):
        self.server = None
        self.grpc_servers = []

        try:
            self.listen_addr = os.environ["AI_ENGINE"]
        except KeyError:
            self.listen_addr = "0.0.0.0:5000"

    def init_grpc_servers(self):
        """
        init_grpc_servers method that initializes and registers gRPC servers
        :return: None
        """
        self.server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
        self.grpc_servers.append(APIClassifierServer())

        grpc_server: GRPCServer
        for grpc_server in self.grpc_servers:
            grpc_server.register(self.server)

    def serve(self):
        """
        serve method that starts serving the gRPC servers (blocking function)
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


class APIClassifierServer(sentryflow_metrics_pb2_grpc.APIClassifierServicer, GRPCServer):
    """
    Class for API Classification Server using Stringlifier
    """
    def __init__(self):
        self.stringlifier = Stringlifier()
        print("[Init] Successfully initialized APIClassificationServer")

    def register(self, server):
        sentryflow_metrics_pb2_grpc.add_APIClassifierServicer_to_server(self, server)

    def ClassifyAPIs(self, request_iterator, _):  # pylint: disable=C0103
        """
        ClassifyAPIs method that runs multiple MLs for API Classification at once
        :param request_iterator: The requests
        :param context: The context
        :return: The results
        """
        for req in request_iterator:
            all_paths = req.API
            ml_results = self.stringlifier(all_paths)

            ml_counts = Counter(ml_results)
            print(f"{all_paths} -> {ml_counts}")

            yield sentryflow_metrics_pb2.APIClassifierResponse(APIs=ml_counts)


if __name__ == '__main__':
    hs = HandlerServer()
    hs.init_grpc_servers()
    hs.serve()
