import React, { Fragment, useCallback, useContext } from 'react';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import Paper from '@mui/material/Paper';
import Popover from '@mui/material/Popover';
import TableCell from '@mui/material/TableCell';
import TableRow from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
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
    Team,
    visibilityString
} from '../api';
import { IconMenuItem, NestedMenuItem } from 'mui-nested-menu';
import SteamID from 'steamid';
import { formatExternalLink, openInNewTab, writeToClipboard } from '../util';
import { SettingsContext } from '../context/settings';
import sb from '../img/sb.png';
import dead from '../img/dead.png';
import vac from '../img/vac.png';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableBody from '@mui/material/TableBody';
import { format, parseJSON } from 'date-fns';
import { TextareaAutosize } from '@mui/material';

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
        teamABg: '#001d49',
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
    if (player.matches && player.matches.length) {
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
            <Grid xs={3} key={`${key}-key`} padding={0}>
                <Typography variant={'button'} textAlign={'right'}>
                    {key}
                </Typography>
            </Grid>,
            <Grid xs={9} key={`${key}-val`} padding={0}>
                <Typography variant={'body1'}>{value}</Typography>
            </Grid>
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
                    <TableCell align={'right'}>
                        <Typography variant={'overline'}>
                            {player.user_id}
                        </Typography>
                    </TableCell>
                )}
                {enabledColumns.includes('name') && (
                    <TableCell>
                        <Grid container spacing={1}>
                            {!player.alive && (
                                <Grid
                                    xs={'auto'}
                                    display="flex"
                                    justifyContent="center"
                                    alignItems="center"
                                >
                                    <img
                                        width={18}
                                        height={18}
                                        src={dead}
                                        alt={`Player is dead (lol)`}
                                    />
                                </Grid>
                            )}
                            <Grid xs textOverflow={'clip'} overflow={'hidden'}>
                                <Typography
                                    sx={{
                                        fontFamily: 'Monospace',
                                        maxWidth: '250px'
                                    }}
                                    variant={'subtitle1'}
                                >
                                    {player.name}
                                </Typography>
                            </Grid>

                            {player.number_of_vac_bans > 0 && (
                                <Grid
                                    xs={'auto'}
                                    display="flex"
                                    justifyContent="center"
                                    alignItems="center"
                                >
                                    <img
                                        width={18}
                                        height={18}
                                        src={vac}
                                        alt={`${player.number_of_vac_bans} VAC bans on record`}
                                    />
                                </Grid>
                            )}
                            {player.sourcebans &&
                                player.sourcebans.length > 0 && (
                                    <Grid
                                        xs={'auto'}
                                        display="flex"
                                        justifyContent="center"
                                        alignItems="center"
                                    >
                                        <img
                                            width={18}
                                            height={18}
                                            src={sb}
                                            alt={`${player.sourcebans.length} Sourcebans entries on record`}
                                        />
                                    </Grid>
                                )}
                        </Grid>
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
                            {player.alive ? player.health : 0}
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
                <Paper style={{ maxWidth: 650 }} sx={{ padding: 1 }}>
                    <Grid container spacing={1}>
                        <Grid xs={'auto'}>
                            <img
                                height={184}
                                width={184}
                                alt={player.name}
                                src={`https://avatars.cloudflare.steamstatic.com/${player.avatar_hash}_full.jpg`}
                            />
                        </Grid>
                        <Grid xs>
                            <div>
                                <Grid container>
                                    {...makeInfoRow(
                                        'UID',
                                        player.user_id.toString()
                                    )}
                                    {...makeInfoRow('Name', player.name)}
                                    {...makeInfoRow(
                                        'Profile Visibility',
                                        visibilityString(player.visibility)
                                    )}
                                    {...makeInfoRow(
                                        'Vac Bans',
                                        player.number_of_vac_bans.toString()
                                    )}
                                    {...makeInfoRow(
                                        'Game Bans',
                                        player.number_of_game_bans.toString()
                                    )}
                                </Grid>
                            </div>
                        </Grid>
                        {player.notes.length > 0 && (
                            <Grid xs={12}>
                                <TextareaAutosize value={player.notes} />
                            </Grid>
                        )}
                        {player.matches && player.matches.length > 0 && (
                            <Grid xs={12}>
                                <TableContainer>
                                    <TableHead>
                                        <TableRow>
                                            <TableCell padding={'normal'}>
                                                Origin
                                            </TableCell>
                                            <TableCell padding={'normal'}>
                                                Tags
                                            </TableCell>
                                            <TableCell padding={'normal'}>
                                                Type
                                            </TableCell>
                                        </TableRow>
                                    </TableHead>
                                    <TableBody>
                                        {player.matches?.map((match) => {
                                            return (
                                                <TableRow
                                                    key={`match-${match.origin}`}
                                                >
                                                    <TableCell>
                                                        <Typography
                                                            padding={1}
                                                            variant={'button'}
                                                        >
                                                            {match.origin}
                                                        </Typography>
                                                    </TableCell>
                                                    <TableCell>
                                                        <Typography
                                                            padding={1}
                                                            variant={'button'}
                                                        >
                                                            {match.attributes.join(
                                                                ', '
                                                            )}
                                                        </Typography>
                                                    </TableCell>
                                                    <TableCell>
                                                        <Typography
                                                            padding={1}
                                                            variant={'button'}
                                                        >
                                                            {match.matcher_type}
                                                        </Typography>
                                                    </TableCell>
                                                </TableRow>
                                            );
                                        })}
                                    </TableBody>
                                </TableContainer>
                            </Grid>
                        )}
                        {player.sourcebans && player.sourcebans.length > 0 && (
                            <Grid xs={12}>
                                <TableContainer>
                                    <TableHead>
                                        <TableRow>
                                            <TableCell padding={'normal'}>
                                                Site&nbsp;Name
                                            </TableCell>
                                            <TableCell padding={'normal'}>
                                                Created
                                            </TableCell>
                                            <TableCell padding={'normal'}>
                                                Perm
                                            </TableCell>
                                            <TableCell
                                                padding={'normal'}
                                                width={'100%'}
                                            >
                                                Reason
                                            </TableCell>
                                        </TableRow>
                                    </TableHead>
                                    <TableBody>
                                        {player.sourcebans.map((ban) => {
                                            return (
                                                <TableRow
                                                    key={`sb-${ban.ban_id}`}
                                                >
                                                    <TableCell>
                                                        <Typography
                                                            padding={1}
                                                            variant={'button'}
                                                        >
                                                            {ban.site_name}
                                                        </Typography>
                                                    </TableCell>
                                                    <TableCell>
                                                        <Typography
                                                            padding={1}
                                                            variant={'button'}
                                                        >
                                                            {format(
                                                                parseJSON(
                                                                    ban.created_on
                                                                ),
                                                                'MM/dd/yyyy'
                                                            )}
                                                        </Typography>
                                                    </TableCell>
                                                    <TableCell>
                                                        <Typography
                                                            padding={1}
                                                            variant={'button'}
                                                        >
                                                            {ban.permanent
                                                                ? 'yes'
                                                                : 'no'}
                                                        </Typography>
                                                    </TableCell>
                                                    <TableCell>
                                                        <Typography
                                                            padding={1}
                                                            variant={'body1'}
                                                        >
                                                            {ban.reason}
                                                        </Typography>
                                                    </TableCell>
                                                </TableRow>
                                            );
                                        })}
                                    </TableBody>
                                </TableContainer>
                            </Grid>
                        )}
                    </Grid>
                </Paper>
            </Popover>
        </Fragment>
    );
};
