import { createLazyFileRoute } from '@tanstack/react-router';
import React, { Fragment } from 'react';

export const Route = createLazyFileRoute('/pagenotfound')({
    component: PageNotfound
});

function PageNotfound() {
    return <Fragment>not found</Fragment>;
}
