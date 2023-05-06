import React, { Fragment } from 'react';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import Paper from '@mui/material/Paper';
import Popover from '@mui/material/Popover';
import Stack from '@mui/material/Stack';
import TableCell from '@mui/material/TableCell';
import TableRow from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import Grid2 from '@mui/material/Unstable_Grid2';
import FlagIcon from '@mui/icons-material/Flag';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import LinkOutlinedIcon from '@mui/icons-material/LinkOutlined';
import ContentCopyOutlinedIcon from '@mui/icons-material/ContentCopyOutlined';
import ForumOutlinedIcon from '@mui/icons-material/ForumOutlined';
import BadgeOutlinedIcon from '@mui/icons-material/BadgeOutlined';
import NotificationsPausedOutlinedIcon from '@mui/icons-material/NotificationsPausedOutlined';
import NoteAltOutlinedIcon from '@mui/icons-material/NoteAltOutlined';
import { validColumns } from './PlayerTable';
import { formatSeconds, Player, Team } from '../api';

export interface TableRowContextMenuProps {
    enabledColumns: validColumns[];
    player: Player;
}

interface userTheme {
    connectingBg: string;
    matchCheaterBg: string;
    matchBotBg: string;
    matchOtherBg: string;
    vacBansBg: string;
    gameBansBg: string;
    teamABg: string;
    teamBBg: string;
}

const createUserTheme = (): userTheme => {
    // TODO user configurable
    return {
        connectingBg: '#032a23',
        teamABg: '#062c15',
        matchBotBg: '#901380',
        matchCheaterBg: '#500e0e',
        matchOtherBg: '#0c1341',
        teamBBg: '#032a23',
        gameBansBg: '#383615',
        vacBansBg: '#55521f'
    };
};

const curTheme = createUserTheme();

export const rowColour = (player: Player): string => {
    if (player.matches.length) {
        if (
            player.matches.filter((m) => {
                m.attributes.includes('cheater');
            })
        ) {
            return curTheme.matchCheaterBg;
        } else if (
            player.matches.filter((m) => {
                m.attributes.includes('bot');
            })
        ) {
            return curTheme.matchBotBg;
        }
        return curTheme.matchOtherBg;
    } else if (player.number_of_vac_bans) {
        return curTheme.vacBansBg;
    } else if (player.number_of_game_bans) {
        return curTheme.gameBansBg;
    } else if (player.team == Team.RED) {
        return curTheme.teamABg;
    } else if (player.team == Team.BLU) {
        return curTheme.teamBBg;
    }
    return curTheme.connectingBg;
};

export const TableRowContextMenu = ({
    player,
    enabledColumns
}: TableRowContextMenuProps): JSX.Element => {
    //const [anchorEl, setAnchorEl] = useState<HTMLTableRowElement | null>(null);
    //const [contextMenu, setContextMenu] = useState<ContextMenu>(null);

    const [hoverMenuPos, setHoverMenuPos] = React.useState<{
        mouseX: number;
        mouseY: number;
    } | null>(null);

    const [contextMenuPos, setContextMenuPos] = React.useState<{
        mouseX: number;
        mouseY: number;
    } | null>(null);

    const handleRowClick = (event: React.MouseEvent<HTMLTableRowElement>) => {
        setContextMenuPos(
            contextMenuPos === null
                ? {
                      mouseX: event.clientX + 2,
                      mouseY: event.clientY - 6
                  }
                : // repeated contextmenu when it is already open closes it with Chrome 84 on Ubuntu
                  // Other native context menus might behave different.
                  // With this behavior we prevent contextmenu from the backdrop to re-locale existing context menus.
                  null
        );
    };

    const handleMenuClose = () => {
        setContextMenuPos(null);
    };

    const mouseEnter = (event: React.MouseEvent<HTMLTableRowElement>) => {
        setHoverMenuPos(
            contextMenuPos === null
                ? {
                      mouseX: event.clientX + 2,
                      mouseY: event.clientY - 6
                  }
                : // repeated contextmenu when it is already open closes it with Chrome 84 on Ubuntu
                  // Other native context menus might behave different.
                  // With this behavior we prevent contextmenu from the backdrop to re-locale existing context menus.
                  null
        );
    };

    const mouseLeave = () => {
        setHoverMenuPos(null);
    };

    const makeInfoRow = (key: string, value: any): JSX.Element[] => {
        return [
            <Grid2 xs={3} key={`${key}-key`} padding={0}>
                <Typography variant={'button'} textAlign={'right'}>
                    {key}
                </Typography>
            </Grid2>,
            <Grid2 xs={9} key={`${key}-val`} padding={0}>
                <Typography variant={'button'}>{value}</Typography>
            </Grid2>
        ];
    };

    return (
        <Fragment key={`${player.steam_id}`}>
            <TableRow
                hover
                //onClick={handleClick}
                //onContextMenu={handleContextMenu}
                style={{
                    backgroundColor: rowColour(player),
                    cursor: 'pointer'
                }}
                key={`row-${player.steam_id}`}
                onMouseEnter={mouseEnter}
                onMouseLeave={mouseLeave}
                onClick={handleRowClick}
                sx={{
                    '&:hover': {
                        backgroundColor: 'primary'
                    }
                }}
            >
                {enabledColumns.includes('user_id') && (
                    <TableCell align={'right'} style={{ paddingRight: 6 }}>
                        <Typography variant={'overline'}>
                            {player.user_id}
                        </Typography>
                    </TableCell>
                )}
                {enabledColumns.includes('name') && (
                    <TableCell>
                        <Typography
                            sx={{ fontFamily: 'Monospace' }}
                            overflow={'ellipsis'}
                        >
                            {player.name}
                        </Typography>
                    </TableCell>
                )}
                {enabledColumns.includes('kills') && (
                    <TableCell align={'right'}>
                        <Typography variant={'overline'}>
                            {player.kills}
                        </Typography>
                    </TableCell>
                )}
                {enabledColumns.includes('deaths') && (
                    <TableCell align={'right'}>
                        <Typography variant={'overline'}>
                            {player.deaths}
                        </Typography>
                    </TableCell>
                )}
                {enabledColumns.includes('connected') && (
                    <TableCell align={'right'}>
                        <Typography variant={'overline'}>
                            {formatSeconds(player.connected)}
                        </Typography>
                    </TableCell>
                )}
                {enabledColumns.includes('ping') && (
                    <TableCell align={'right'} style={{ paddingRight: 6 }}>
                        <Typography variant={'overline'}>
                            {' '}
                            {player.ping}
                        </Typography>
                    </TableCell>
                )}
            </TableRow>
            <Menu
                open={contextMenuPos !== null}
                onClose={handleMenuClose}
                anchorReference="anchorPosition"
                anchorPosition={
                    contextMenuPos !== null
                        ? {
                              top: contextMenuPos.mouseY,
                              left: contextMenuPos.mouseX
                          }
                        : undefined
                }
            >
                <MenuItem disableRipple>
                    <img
                        alt={`Avatar`}
                        src={`https://avatars.cloudflare.steamstatic.com/${player.avatar_hash}_full.jpg`}
                    />
                </MenuItem>
                <MenuItem>
                    <ListItemIcon>
                        <FlagIcon color={'primary'} />
                    </ListItemIcon>
                    <ListItemText>Mark As...</ListItemText>
                </MenuItem>
                <MenuItem>
                    <ListItemIcon>
                        <DeleteOutlinedIcon color={'primary'} />
                    </ListItemIcon>
                    <ListItemText>Unmark</ListItemText>
                </MenuItem>
                <MenuItem>
                    <ListItemIcon>
                        <LinkOutlinedIcon color={'primary'} />
                    </ListItemIcon>
                    <ListItemText>Open External</ListItemText>
                </MenuItem>
                <MenuItem>
                    <ListItemIcon>
                        <ContentCopyOutlinedIcon color={'primary'} />
                    </ListItemIcon>
                    <ListItemText>Copy SteamID</ListItemText>
                </MenuItem>
                <MenuItem>
                    <ListItemIcon>
                        <ForumOutlinedIcon color={'primary'} />
                    </ListItemIcon>
                    <ListItemText>Chat History</ListItemText>
                </MenuItem>
                <MenuItem>
                    <ListItemIcon>
                        <BadgeOutlinedIcon color={'primary'} />
                    </ListItemIcon>
                    <ListItemText>Name History</ListItemText>
                </MenuItem>
                <MenuItem>
                    <ListItemIcon>
                        <NotificationsPausedOutlinedIcon color={'primary'} />
                    </ListItemIcon>
                    <ListItemText>Whitelist</ListItemText>
                </MenuItem>
                <MenuItem>
                    <ListItemIcon>
                        <NoteAltOutlinedIcon color={'primary'} />
                    </ListItemIcon>
                    <ListItemText>Edit Notes</ListItemText>
                </MenuItem>
            </Menu>
            <Popover
                open={hoverMenuPos !== null}
                sx={{
                    pointerEvents: 'none'
                }}
                anchorReference="anchorPosition"
                anchorPosition={
                    hoverMenuPos !== null
                        ? {
                              top: hoverMenuPos.mouseY,
                              left: hoverMenuPos.mouseX
                          }
                        : undefined
                }
                disablePortal={true}
                //disableRestoreFocus
            >
                <Paper style={{ maxWidth: 650 }}>
                    <Stack padding={1} direction={'row'} spacing={1}>
                        <img
                            height={184}
                            width={184}
                            alt={player.name}
                            src={`https://avatars.cloudflare.steamstatic.com/${player.avatar_hash}_full.jpg`}
                        />
                        <Grid2 container>
                            {...makeInfoRow('UID', player.user_id)}
                            {...makeInfoRow('Name', player.name)}
                            {...makeInfoRow('Kills', player.kills.toString())}
                            {...makeInfoRow('Deaths', player.deaths.toString())}
                            {...makeInfoRow('Time', player.connected)}
                            {...makeInfoRow('Ping', player.ping.toString())}
                            {...makeInfoRow(
                                'Vac Bans',
                                player.number_of_vac_bans.toString()
                            )}
                            {...makeInfoRow(
                                'Game Bans',
                                player.number_of_game_bans.toString()
                            )}
                        </Grid2>
                    </Stack>
                </Paper>
            </Popover>
        </Fragment>
    );
};
