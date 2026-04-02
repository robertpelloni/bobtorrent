#pragma once

#include "apicontroller.h"

class MegatorrentController : public APIController
{
    Q_OBJECT
    Q_DISABLE_COPY_MOVE(MegatorrentController)

public:
    explicit MegatorrentController(IApplication *app, QObject *parent = nullptr);

    // REST-like API methods (exposed via QObject slots)
    Q_INVOKABLE void statusAction();
    Q_INVOKABLE void generateKeyAction();
    Q_INVOKABLE void ingestAction();
    Q_INVOKABLE void publishAction();
    Q_INVOKABLE void subscribeAction();
    Q_INVOKABLE void subscriptionsAction();
    Q_INVOKABLE void unsubscribeAction();
    Q_INVOKABLE void blobsAction();
};
