import React, { useCallback, useContext, useMemo } from 'react';
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
import { addWhitelist, Player, useCurrentState } from '../api';
import ViewColumnIcon from '@mui/icons-material/ViewColumn';
import { Trans } from 'react-i18next';
import { logError } from '../util';
import { PlayerTableRow } from './PlayerTableRow';
import { PlayerTableContext } from '../context/PlayerTableContext';

const descendingComparator = <T,>(a: T, b: T, orderBy: keyof T) => {
    if (b[orderBy] < a[orderBy]) {
        return -1;
    }
    if (b[orderBy] > a[orderBy]) {
        return 1;
    }
    return 0;
};

export type Order = 'asc' | 'desc';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const getComparator = <Key extends keyof any>(
    order: Order,
    orderBy: Key
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
): ((a: { [key in Key]: any }, b: { [key in Key]: any }) => number) =>
    order === 'asc'
        ? (a, b) => descendingComparator(a, b, orderBy)
        : (a, b) => -descendingComparator(a, b, orderBy);

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const stableSort = <T,>(
    array: readonly T[],
    comparator: (a: T, b: T) => number
) => {
    const stabilizedThis = array.map((el, index) => [el, index] as [T, number]);
    stabilizedThis.sort((a, b) => {
        const order = comparator(a[0], b[0]);
        if (order !== 0) {
            return order;
        }
        return a[1] - b[1];
    });
    return stabilizedThis.map((el) => el[0]);
};

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
    const [anchorEl, setAnchorEl] = React.useState<HTMLButtonElement | null>(
        null
    );

    const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
        setAnchorEl(event.currentTarget);
    };

    const handleClose = () => {
        setAnchorEl(null);
    };

    const handleColumnsChange = (
        _: React.MouseEvent<HTMLElement>,
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
    const createSortHandler = (property: keyof Player) => () => {
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
    const state = useCurrentState();
    const { orderBy, order, matchesOnly } = useContext(PlayerTableContext);
    const onWhitelist = useCallback(async (steamId: string) => {
        try {
            await addWhitelist(steamId);
        } catch (e) {
            logError(`Error adding whitelist: ${e}`);
        }
    }, []);

    const visibleRows = useMemo(() => {
        const filteredPlayers = state.players.filter(
            (p) => !matchesOnly || (!p.whitelisted && p.matches?.length)
        );
        return stableSort(filteredPlayers, getComparator(order, orderBy));
    }, [order, orderBy, state.players, matchesOnly]);

    const playerRows = useMemo(() => {
        return visibleRows.map((player, i) => (
            <PlayerTableRow
                onWhitelist={onWhitelist}
                player={player}
                key={`player-row-${i}-${player.steam_id}`}
            />
        ));
    }, [onWhitelist, visibleRows]);

    return (
        <TableContainer sx={{ overflow: 'hidden' }}>
            <Table aria-label="Player table" size="small" padding={'none'}>
                <PlayerTableHead />
                <TableBody>{playerRows}</TableBody>
            </Table>
        </TableContainer>
    );
};
