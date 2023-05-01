import React, { useCallback, useEffect, useMemo, useState } from 'react';

import {
    Paper,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow
} from '@mui/material';
import { getPlayers, Player } from '../api';
import { TableRowContextMenu } from './ContextMenu';
type sortableColumns = 'name' | 'kills' | 'ping' | 'status';

export const PlayerTable = () => {
    const [players, setPlayers] = useState<Player[]>([]);
    const [sortBy, setSortBy] = useState<sortableColumns>('kills');
    const [sortDesc, setSortDesc] = useState(false);

    useEffect(() => {
        const interval = setInterval(async () => {
            try {
                setPlayers(await getPlayers());
            } catch (e) {
                console.log(e);
            }
        }, 2 * 1000);
        return () => {
            clearInterval(interval);
        };
    }, []);

    const sortedPlayers = useMemo(() => {
        const newlySorted = players;
        newlySorted.sort((a: Player, b: Player) => {
            switch (sortBy) {
                case 'kills':
                    return sortDesc ? a.kills - b.kills : b.kills - a.kills;
                case 'ping':
                    return sortDesc ? a.ping - b.ping : b.ping - a.ping;
                case 'name':
                    return sortDesc
                        ? a.name.localeCompare(b.name)
                        : b.name.localeCompare(a.name);
                default:
                    return sortDesc ? a.kills - b.kills : b.kills - a.kills;
            }
        });
        return newlySorted;
    }, [sortBy, sortDesc, players]);

    const updateSortDir = useCallback((columnName: sortableColumns) => {
        if (columnName == sortBy) {
            setSortDesc(function (prevState) {
                const ns = !prevState;
                console.log(`Set sort: ${columnName} ${ns}`);
                return ns;
            });
            return;
        }
        setSortBy(columnName);
    }, []);

    const mkHeaderCell = (columnName: sortableColumns) => {
        return (
            <TableCell
                key={`header-${columnName}`}
                onClick={() => {
                    updateSortDir(columnName);
                }}
            >
                {columnName.charAt(0).toUpperCase() + columnName.slice(1)}
            </TableCell>
        );
    };
    const columns: sortableColumns[] = ['name', 'kills', 'ping', 'status'];

    return (
        <Paper sx={{ width: '100%', overflow: 'hidden' }}>
            <TableContainer>
                <Table
                    stickyHeader
                    aria-label="sticky table"
                    size="small"
                    padding={'none'}
                >
                    <TableHead>
                        <TableRow>{columns.map(mkHeaderCell)}</TableRow>
                    </TableHead>

                    <TableBody>
                        {sortedPlayers.map((player, i) => (
                            <TableRowContextMenu
                                player={player}
                                key={`row-${i}-${player.steam_id}`}
                            />
                        ))}
                    </TableBody>
                </Table>
            </TableContainer>
        </Paper>
    );
};
