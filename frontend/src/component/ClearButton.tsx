import { Button } from '@mui/material';
import ClearIcon from '@mui/icons-material/Clear';
import { Trans } from 'react-i18next';
import { onClickProps } from '../util.ts';

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

export default ClearButton;
