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
