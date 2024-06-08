import { useCallback, useContext, useMemo, useState, MouseEvent } from 'react';
import Box from '@mui/material/Box';
import IconButton from '@mui/material/IconButton';
import Paper from '@mui/material/Paper';
import Popover from '@mui/material/Popover';
import Stack from '@mui/material/Stack';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import TableSortLabel from '@mui/material/TableSortLabel';
import ToggleButton from '@mui/material/ToggleButton';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import Tooltip from '@mui/material/Tooltip';
import ViewColumnIcon from '@mui/icons-material/ViewColumn';
import { Trans } from 'react-i18next';
import { PlayerTableRow } from './PlayerTableRow';
import { PlayerTableContext } from '../context/PlayerTableContext';
import { useGameState } from '../context/GameStateContext.ts';
import { useMutation } from '@tanstack/react-query';
import { addWhitelistMutation } from '../api.ts';

export type Order = 'asc' | 'desc';

interface HeadCell {
    disablePadding: boolean;
    id: validColumns;
    label: string;
    numeric: boolean;
    tooltip: string;
}

export type validColumns =
    | 'user_id'
    | 'name'
    | 'score'
    | 'kills'
    | 'deaths'
    | 'kpm'
    | 'connected'
    | 'map_time'
    | 'ping'
    | 'health'
    | 'alive';

const headCells: readonly HeadCell[] = [
    {
        id: 'user_id',
        numeric: true,
        disablePadding: false,
        label: 'uid',
        tooltip: 'Players in-Game user id'
    },
    {
        id: 'name',
        numeric: false,
        disablePadding: false,
        label: 'name',
        tooltip: 'Players current name, as reported by the game server'
    },
    {
        id: 'score',
        numeric: true,
        disablePadding: false,
        label: 'score',
        tooltip: 'Players current score'
    },
    {
        id: 'kills',
        numeric: true,
        disablePadding: false,
        label: 'kills',
        tooltip: 'Players current kills'
    },
    {
        id: 'deaths',
        numeric: true,
        disablePadding: false,
        label: 'deaths',
        tooltip: 'Players current deaths'
    },
    {
        id: 'kpm',
        numeric: true,
        disablePadding: false,
        label: 'kpm',
        tooltip:
            'Players kills per minute. Calculated from when you first see the player in the server, not how long they have actually been in the server'
    },
    {
        id: 'health',
        numeric: true,
        disablePadding: false,
        label: 'health',
        tooltip: 'Shows player current health'
    },
    {
        id: 'connected',
        numeric: true,
        disablePadding: false,
        label: 'time',
        tooltip: 'How long the player has been connected to the server'
    },
    {
        id: 'map_time',
        numeric: true,
        disablePadding: false,
        label: 'map time',
        tooltip: 'How long ist been since you first joined the map'
    },
    {
        id: 'ping',
        numeric: true,
        disablePadding: false,
        label: 'ping',
        tooltip: 'Players current latency'
    }
];

export const ColumnConfigButton = () => {
    const { saveSelectedColumns, enabledColumns } =
        useContext(PlayerTableContext);
    const [anchorEl, setAnchorEl] = useState<HTMLButtonElement | null>(null);

    const handleClick = (event: MouseEvent<HTMLButtonElement>) => {
        setAnchorEl(event.currentTarget);
    };

    const handleClose = () => {
        setAnchorEl(null);
    };

    const handleColumnsChange = (
        _: MouseEvent<HTMLElement>,
        newFormats: validColumns[]
    ) => {
        saveSelectedColumns(newFormats);
    };

    const open = Boolean(anchorEl);
    return (
        <>
            <IconButton onClick={handleClick}>
                <ViewColumnIcon color={'primary'} />
            </IconButton>
            <Popover
                open={open}
                onClose={handleClose}
                id={'column-config-popover'}
                anchorEl={anchorEl}
                anchorOrigin={{
                    vertical: 'bottom',
                    horizontal: 'left'
                }}
            >
                <Paper>
                    <Stack>
                        <ToggleButtonGroup
                            color="primary"
                            orientation={'vertical'}
                            value={enabledColumns}
                            onChange={handleColumnsChange}
                            aria-label="Visible Columns"
                        >
                            {headCells.map((r) => {
                                return (
                                    <ToggleButton
                                        name={r.label}
                                        value={r.id}
                                        key={`column-toggle-${r.id}`}
                                    >
                                        <Trans
                                            i18nKey={`player_table.column.${r.id}`}
                                        />
                                    </ToggleButton>
                                );
                            })}
                        </ToggleButtonGroup>
                    </Stack>
                </Paper>
            </Popover>
        </>
    );
};

const PlayerTableHead = () => {
    const { enabledColumns, orderBy, order, saveSortColumn } =
        useContext(PlayerTableContext);
    const createSortHandler = (property: validColumns) => () => {
        saveSortColumn(property);
    };

    return (
        <TableHead>
            <TableRow>
                {headCells
                    .filter(
                        (c) => enabledColumns.includes(c.id) || !enabledColumns
                    )
                    .map((headCell) => (
                        <Tooltip title={headCell.tooltip} key={headCell.id}>
                            <TableCell
                                align={headCell.numeric ? 'right' : 'left'}
                                padding={
                                    headCell.disablePadding ? 'none' : 'normal'
                                }
                                sortDirection={
                                    orderBy === headCell.id ? order : false
                                }
                            >
                                <TableSortLabel
                                    active={orderBy === headCell.id}
                                    direction={
                                        orderBy === headCell.id ? order : 'asc'
                                    }
                                    onClick={createSortHandler(headCell.id)}
                                >
                                    <Trans
                                        i18nKey={`player_table.column.${headCell.id}`}
                                    />
                                    {orderBy === headCell.id ? (
                                        <Box
                                            component="span"
                                            sx={{ display: 'none' }}
                                        >
                                            {order === 'desc'
                                                ? 'sorted descending'
                                                : 'sorted ascending'}
                                        </Box>
                                    ) : null}
                                </TableSortLabel>
                            </TableCell>
                        </Tooltip>
                    ))}
            </TableRow>
        </TableHead>
    );
};

export const PlayerTable = () => {
    const { state } = useGameState();

    const whitelist = useMutation(addWhitelistMutation());

    const onWhitelist = useCallback(
        async (steamId: string) => {
            whitelist.mutate({ steamId });
        },
        [whitelist.mutate]
    );

    const playerRows = useMemo(() => {
        return state.players.map((player, i) => (
            <PlayerTableRow
                onWhitelist={onWhitelist}
                player={player}
                key={`player-row-${i}-${player.steam_id}`}
            />
        ));
    }, [onWhitelist]);

    return (
        <TableContainer sx={{ overflow: 'hidden' }}>
            <Table aria-label="Player table" size="small" padding={'none'}>
                <PlayerTableHead />
                <TableBody>{playerRows}</TableBody>
            </Table>
        </TableContainer>
    );
};
