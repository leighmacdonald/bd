import { createContext, Dispatch, SetStateAction, useContext } from 'react';
import { GameState } from '../api';
import { noop } from '../util.ts';

export const defaultState: GameState = {
    game_running: false,
    server: {
        server_name: '',
        current_map: '',
        tags: [],
        last_update: ''
    },
    players: []
};

interface StateContextProps {
    state: GameState;
    setState: Dispatch<SetStateAction<GameState>>;
}

export const GameStateContext = createContext<StateContextProps>({
    state: defaultState,
    setState: noop
});

export const useGameState = () => useContext(GameStateContext);
