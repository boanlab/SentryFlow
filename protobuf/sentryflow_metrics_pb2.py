# -*- coding: utf-8 -*-
# Generated by the protocol buffer compiler.  DO NOT EDIT!
# source: sentryflow_metrics.proto
# Protobuf Python Version: 5.26.1
"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()




DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n\x18sentryflow_metrics.proto\x12\x08protobuf\"#\n\x14\x41PIClassifierRequest\x12\x0b\n\x03\x41PI\x18\x01 \x03(\t\"}\n\x15\x41PIClassifierResponse\x12\x37\n\x04\x41PIs\x18\x01 \x03(\x0b\x32).protobuf.APIClassifierResponse.APIsEntry\x1a+\n\tAPIsEntry\x12\x0b\n\x03key\x18\x01 \x01(\t\x12\r\n\x05value\x18\x02 \x01(\x04:\x02\x38\x01\x32\x64\n\rAPIClassifier\x12S\n\x0c\x43lassifyAPIs\x12\x1e.protobuf.APIClassifierRequest\x1a\x1f.protobuf.APIClassifierResponse(\x01\x30\x01\x42\x15Z\x13SentryFlow/protobufb\x06proto3')

_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'sentryflow_metrics_pb2', _globals)
if not _descriptor._USE_C_DESCRIPTORS:
  _globals['DESCRIPTOR']._loaded_options = None
  _globals['DESCRIPTOR']._serialized_options = b'Z\023SentryFlow/protobuf'
  _globals['_APICLASSIFIERRESPONSE_APISENTRY']._loaded_options = None
  _globals['_APICLASSIFIERRESPONSE_APISENTRY']._serialized_options = b'8\001'
  _globals['_APICLASSIFIERREQUEST']._serialized_start=38
  _globals['_APICLASSIFIERREQUEST']._serialized_end=73
  _globals['_APICLASSIFIERRESPONSE']._serialized_start=75
  _globals['_APICLASSIFIERRESPONSE']._serialized_end=200
  _globals['_APICLASSIFIERRESPONSE_APISENTRY']._serialized_start=157
  _globals['_APICLASSIFIERRESPONSE_APISENTRY']._serialized_end=200
  _globals['_APICLASSIFIER']._serialized_start=202
  _globals['_APICLASSIFIER']._serialized_end=302
# @@protoc_insertion_point(module_scope)
