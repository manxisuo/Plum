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
PKGCONFIG += grpc++ protobuf

LIBS += -L$$PWD/../../sdk/cpp/build
LIBS += -L$$PWD/../../sdk/cpp/build/plumworker
LIBS += -L$$PWD/../../sdk/cpp/build/grpc_proto
LIBS += -lplumworker -lgrpc_proto -ldl

