import React, { useCallback, useState } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import { Order, PlayerTable, validColumns } from '../component/PlayerTable.tsx';
import Paper from '@mui/material/Paper';

import Stack from '@mui/material/Stack';
import { Toolbar } from '../component/Toolbar.tsx';
import { Player } from '../api.ts';
import { PlayerTableContext } from '../context/PlayerTableContext.ts';
import { getDefaultColumns } from '../table.ts';
import { createLazyFileRoute } from '@tanstack/react-router';

export const Route = createLazyFileRoute('/')({
    component: Index
});

function Index() {
    const [order, setOrder] = useState<Order>(
        (localStorage.getItem('sortOrder') as Order) ?? 'desc'
    );
    const [orderBy, setOrderBy] = useState<keyof Player>(
        (localStorage.getItem('sortBy') as keyof Player) ?? 'personaname'
    );
    const [matchesOnly, setMatchesOnly] = useState(
        JSON.parse(localStorage.getItem('matchesOnly') || 'false') === true
    );
    const [enabledColumns, setEnabledColumns] =
        useState<validColumns[]>(getDefaultColumns());

    const saveSelectedColumns = useCallback(
        (columns: validColumns[]) => {
            setEnabledColumns(columns);
            localStorage.setItem('enabledColumns', JSON.stringify(columns));
        },
        [setEnabledColumns]
    );

    const saveSortColumn = useCallback(
        (property: keyof Player) => {
            const isAsc = orderBy === property && order === 'asc';
            const newOrder = isAsc ? 'desc' : 'asc';
            setOrder(newOrder);
            setOrderBy(property);
            localStorage.setItem('sortOrder', newOrder);
            localStorage.setItem('sortBy', property);
        },
        [order, orderBy]
    );

    return (
        <Grid container>
            <Grid xs={12}>
                {/*<SettingsEditorModal id={ModalSettings} />*/}
                <Paper sx={{ width: '100%', overflow: 'hidden' }}>
                    <PlayerTableContext.Provider
                        value={{
                            order,
                            orderBy,
                            setOrderBy,
                            setOrder,
                            setMatchesOnly,
                            matchesOnly,
                            enabledColumns,
                            saveSelectedColumns,
                            saveSortColumn
                        }}
                    >
                        <Stack>
                            <Toolbar />
                            <PlayerTable />
                        </Stack>
                    </PlayerTableContext.Provider>
                </Paper>
            </Grid>
        </Grid>
    );
}
