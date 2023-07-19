import React, { Fragment, useCallback, useContext } from 'react';
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
import ArrowRightOutlinedIcon from '@mui/icons-material/ArrowRightOutlined';
import { validColumns } from './PlayerTable';
import {
    addWhitelist,
    deleteWhitelist,
    formatSeconds,
    Player,
    Team
} from '../api';
import { IconMenuItem, NestedMenuItem } from 'mui-nested-menu';
import SteamID from 'steamid';
import { formatExternalLink, openInNewTab, writeToClipboard } from '../util';
import { SettingsContext } from '../context/settings';

export interface TableRowContextMenuProps {
    enabledColumns: validColumns[];
    player: Player;
    onOpenNotes: (steamId: string, notes: string) => void;
    onSaveNotes: (steamId: string, notes: string) => void;
    onWhitelist: (steamId: string) => void;
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
        teamABg: '#002a84',
        matchBotBg: '#901380',
        matchCheaterBg: '#500e0e',
        matchOtherBg: '#0c1341',
        teamBBg: '#3e020e',
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
    enabledColumns,
    onOpenNotes
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

    const { settings, loading } = useContext(SettingsContext);

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
                : null
        );
    };

    const onDeleteWhitelist = useCallback(async (steamId: string) => {
        await deleteWhitelist(steamId);
    }, []);

    const onAddWhitelist = useCallback(async (steamId: string) => {
        await addWhitelist(steamId);
    }, []);

    const mouseLeave = () => {
        setHoverMenuPos(null);
    };

    const makeInfoRow = (key: string, value: string): JSX.Element[] => {
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
    if (loading || !settings) {
        return <></>;
    }
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
                            {player.alive
                                ? player.name
                                : `*DEAD* ${player.name}`}
                        </Typography>
                    </TableCell>
                )}
                {enabledColumns.includes('score') && (
                    <TableCell align={'right'}>
                        <Typography variant={'overline'}>
                            {player.score}
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
                {enabledColumns.includes('health') && (
                    <TableCell align={'right'}>
                        <Typography variant={'overline'}>
                            {player.health}
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
                <NestedMenuItem
                    rightIcon={<ArrowRightOutlinedIcon />}
                    leftIcon={<FlagIcon color={'primary'} />}
                    label="Mark Player As"
                    parentMenuOpen={contextMenuPos !== null}
                >
                    {[...settings.unique_tags, 'new'].map((tag) => {
                        return (
                            <IconMenuItem
                                leftIcon={<FlagIcon color={'primary'} />}
                                onClick={() => {
                                    console.log(`tag as ${tag}`);
                                }}
                                label={tag}
                                key={`tag-${player.steam_id}-${tag}`}
                            />
                        );
                    })}
                </NestedMenuItem>
                <IconMenuItem
                    leftIcon={<DeleteOutlinedIcon color={'primary'} />}
                    label={'Unmark'}
                    disabled
                />
                <NestedMenuItem
                    rightIcon={<ArrowRightOutlinedIcon />}
                    leftIcon={<LinkOutlinedIcon color={'primary'} />}
                    label="Open External Link"
                    parentMenuOpen={contextMenuPos !== null}
                >
                    {settings.links.map((l) => (
                        <IconMenuItem
                            leftIcon={<FlagIcon color={'primary'} />}
                            onClick={() => {
                                openInNewTab(
                                    formatExternalLink(player.steam_id, l)
                                );
                            }}
                            label={l.name}
                            key={`link-${player.steam_id}-${l.name}`}
                        />
                    ))}
                </NestedMenuItem>
                <NestedMenuItem
                    rightIcon={<ArrowRightOutlinedIcon />}
                    leftIcon={<ContentCopyOutlinedIcon color={'primary'} />}
                    label="Copy SteamID"
                    parentMenuOpen={contextMenuPos !== null}
                >
                    <IconMenuItem
                        leftIcon={<FlagIcon color={'primary'} />}
                        onClick={async () => {
                            await writeToClipboard(
                                new SteamID(
                                    player.steam_id
                                ).getSteam2RenderedID()
                            );
                        }}
                        label={new SteamID(
                            player.steam_id
                        ).getSteam2RenderedID()}
                    />
                    <IconMenuItem
                        leftIcon={<FlagIcon color={'primary'} />}
                        onClick={async () => {
                            await writeToClipboard(
                                new SteamID(
                                    player.steam_id
                                ).getSteam3RenderedID()
                            );
                        }}
                        label={new SteamID(
                            player.steam_id
                        ).getSteam3RenderedID()}
                    />
                    <IconMenuItem
                        leftIcon={<FlagIcon color={'primary'} />}
                        onClick={async () => {
                            await writeToClipboard(
                                new SteamID(player.steam_id).getSteamID64()
                            );
                        }}
                        label={new SteamID(player.steam_id).getSteamID64()}
                    />
                </NestedMenuItem>
                <IconMenuItem
                    leftIcon={<ForumOutlinedIcon color={'primary'} />}
                    label={'Chat History'}
                />
                <IconMenuItem
                    leftIcon={<BadgeOutlinedIcon color={'primary'} />}
                    label={'Name History'}
                />
                {player.whitelisted ? (
                    <IconMenuItem
                        leftIcon={
                            <NotificationsPausedOutlinedIcon
                                color={'primary'}
                            />
                        }
                        label={'Remove Whitelist'}
                        onClick={async () => {
                            await onDeleteWhitelist(player.steam_id);
                        }}
                    />
                ) : (
                    <IconMenuItem
                        leftIcon={
                            <NotificationsPausedOutlinedIcon
                                color={'primary'}
                            />
                        }
                        label={'Whitelist'}
                        onClick={async () => {
                            await onAddWhitelist(player.steam_id);
                        }}
                    />
                )}
                <IconMenuItem
                    leftIcon={<NoteAltOutlinedIcon color={'primary'} />}
                    label={'Edit Notes'}
                    onClick={() => {
                        onOpenNotes(player.steam_id, player.notes);
                        handleMenuClose();
                    }}
                />
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
                disablePortal={false}
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
                            {...makeInfoRow('UID', player.user_id.toString())}
                            {...makeInfoRow('Name', player.name)}
                            {...makeInfoRow('Kills', player.kills.toString())}
                            {...makeInfoRow('Deaths', player.deaths.toString())}
                            {...makeInfoRow(
                                'Time',
                                formatSeconds(player.connected)
                            )}
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
