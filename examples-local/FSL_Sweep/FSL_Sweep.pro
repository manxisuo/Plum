QT += core
CONFIG += console c++17
CONFIG -= app_bundle
TEMPLATE = app

DESTDIR = ../bin
OBJECTS_DIR = build/obj
MOC_DIR = build/moc
RCC_DIR = build/rcc
UI_DIR = build/ui

SOURCES += main.cpp

INCLUDEPATH += $$PWD/../FSL_Common
INCLUDEPATH += $$PWD/include
INCLUDEPATH += $$PWD/../../sdk/cpp/plumworker/include
INCLUDEPATH += $$PWD/../../sdk/cpp/grpc
INCLUDEPATH += $$PWD/../../sdk/cpp/grpc/proto

CONFIG += link_pkgconfig
# 尝试使用静态链接以包含所有依赖（包括 absl）
PKGCONFIG += grpc++ protobuf
# 如果静态链接不工作，可以尝试显式添加 absl 库路径
# PKGCONFIG += grpc++ protobuf absl

LIBS += -L$$PWD/../../sdk/cpp/build
LIBS += -L$$PWD/../../sdk/cpp/build/plumworker
LIBS += -L$$PWD/../../sdk/cpp/build/grpc_proto
LIBS += -lplumworker -lgrpc_proto -ldl

