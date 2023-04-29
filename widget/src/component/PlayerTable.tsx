import React, { useEffect, useState } from 'react';

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

export const PlayerTable = () => {
    const [players, setPlayers] = useState<Player[]>([]);

    useEffect(() => {
        const interval = setInterval(async () => {
            try {
                const newPlayers = await getPlayers();
                setPlayers(newPlayers);
            } catch (e) {
                console.log(e);
            }
        }, 2 * 1000);
        return () => {
            clearInterval(interval);
        };
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
                            <TableCell>Name</TableCell>
                            <TableCell>Kills</TableCell>
                            <TableCell>Ping</TableCell>
                        </TableRow>
                    </TableHead>

                    <TableBody>
                        {players.map((p) => {
                            return (
                                <TableRow hover>
                                    <TableCell>{p.name}</TableCell>
                                    <TableCell>{p.kills}</TableCell>
                                    <TableCell>{p.ping}</TableCell>
                                </TableRow>
                            );
                        })}
                    </TableBody>
                </Table>
            </TableContainer>
        </Paper>
    );
};
