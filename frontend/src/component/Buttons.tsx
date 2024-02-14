import { Trans } from 'react-i18next';
import { Button } from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import SaveIcon from '@mui/icons-material/Save';
import ClearIcon from '@mui/icons-material/Clear';
import RestartAltIcon from '@mui/icons-material/RestartAlt';

interface onClickProps {
    onClick: () => void;
}

export const CancelButton = ({ onClick }: onClickProps) => {
    return (
        <Button
            startIcon={<CloseIcon />}
            color={'error'}
            variant={'contained'}
            onClick={onClick}
        >
            <Trans i18nKey={'button.cancel'} />
        </Button>
    );
};

export const SaveButton = ({
    onClick,
    disabled = false
}: onClickProps & { disabled?: boolean }) => {
    return (
        <Button
            startIcon={<SaveIcon />}
            color={'success'}
            variant={'contained'}
            onClick={onClick}
            disabled={disabled}
        >
            <Trans i18nKey={'button.save'} />
        </Button>
    );
};

export const ClearButton = ({ onClick }: onClickProps) => {
    return (
        <Button
            startIcon={<ClearIcon />}
            color={'warning'}
            variant={'contained'}
            onClick={onClick}
        >
            <Trans i18nKey={'button.clear'} />
        </Button>
    );
};

export const ResetButton = ({ onClick }: onClickProps) => {
    return (
        <Button
            onClick={onClick}
            startIcon={<RestartAltIcon />}
            color={'warning'}
            variant={'contained'}
        >
            <Trans i18nKey={'button.reset'} />
        </Button>
    );
};
