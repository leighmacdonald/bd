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
import { getPlayers, Player, Team } from '../api';

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
                        <TableRow>
                            <TableCell
                                onClick={() => {
                                    updateSortDir('name');
                                }}
                            >
                                Name
                            </TableCell>
                            <TableCell
                                onClick={() => {
                                    updateSortDir('kills');
                                }}
                            >
                                Kills
                            </TableCell>
                            <TableCell
                                onClick={() => {
                                    updateSortDir('ping');
                                }}
                            >
                                Ping
                            </TableCell>
                            <TableCell
                                onClick={() => {
                                    updateSortDir('status');
                                }}
                            >
                                Status
                            </TableCell>
                        </TableRow>
                    </TableHead>

                    <TableBody>
                        {sortedPlayers.map((p) => {
                            let bg = '';
                            if (p.team == Team.BLU) {
                                bg = '#1e2f97';
                            } else if (p.team == Team.RED) {
                                bg = '#179f4d';
                            }
                            let status = '';
                            if (p.number_of_vac_bans) {
                                status += `VAC: ${p.number_of_vac_bans}`;
                                bg = '#383615';
                            }
                            if (p.match != null) {
                                status = `${
                                    p.match.origin
                                } [${p.match.attributes.join(',')}]`;
                                bg = '#500e0e';
                            }
                            return (
                                <TableRow
                                    hover
                                    style={{ backgroundColor: bg }}
                                    key={`row-${p.steam_id}`}
                                >
                                    <TableCell>{p.name}</TableCell>
                                    <TableCell>{p.kills}</TableCell>
                                    <TableCell>{p.ping}</TableCell>
                                    <TableCell>{status}</TableCell>
                                </TableRow>
                            );
                        })}
                    </TableBody>
                </Table>
            </TableContainer>
        </Paper>
    );
};
