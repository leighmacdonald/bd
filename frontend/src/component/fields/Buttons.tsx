import ClearIcon from '@mui/icons-material/Clear';
import CloseIcon from '@mui/icons-material/Close';
import RestartAltIcon from '@mui/icons-material/RestartAlt';
import SendIcon from '@mui/icons-material/Send';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import { t } from 'i18next';

interface ButtonProps {
    canSubmit: boolean;
    isSubmitting: boolean;
    reset: () => void;
    submitLabel?: string;
    resetLabel?: string;
    clearLabel?: string;
    showClear?: boolean;
    showReset?: boolean;
    closeLabel?: string;
    onClear?: () => Promise<void>;
    onClose?: () => Promise<void>;
    fullWidth?: boolean;
}

export const Buttons = ({
    canSubmit,
    isSubmitting,
    reset,
    onClear,
    submitLabel = t('button.save'),
    resetLabel = t('button.reset'),
    clearLabel = t('button.clear'),
    closeLabel = t('button.cancel'),
    showClear = false,
    showReset = true,
    fullWidth = false,
    onClose
}: ButtonProps) => {
    return (
        <ButtonGroup fullWidth={fullWidth}>
            <Button
                key={'submit-button'}
                type="submit"
                disabled={!canSubmit}
                variant={'contained'}
                color={'success'}
                startIcon={<SendIcon />}
            >
                {isSubmitting ? '...' : submitLabel}
            </Button>
            {showReset && (
                <Button
                    key={'reset-button'}
                    type="reset"
                    onClick={() => reset()}
                    variant={'contained'}
                    color={'warning'}
                    startIcon={<RestartAltIcon />}
                >
                    {resetLabel}
                </Button>
            )}
            {showClear ||
                (onClear && (
                    <Button
                        key={'clear-button'}
                        type="button"
                        onClick={async () => {
                            if (onClear) {
                                return await onClear();
                            }
                        }}
                        variant={'contained'}
                        color={'error'}
                        startIcon={<ClearIcon />}
                    >
                        {clearLabel}
                    </Button>
                ))}
            {onClose && (
                <Button
                    key={'close-button'}
                    onClick={onClose}
                    variant={'contained'}
                    color={'error'}
                    startIcon={<CloseIcon />}
                >
                    {closeLabel}
                </Button>
            )}
        </ButtonGroup>
    );
};
