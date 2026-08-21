import { SegmentedControl } from '@mantine/core';
import { useTranslation } from 'react-i18next';

export default function LanguageSwitcher() {
  const { i18n } = useTranslation();
  const lang = i18n.language.startsWith('fr') ? 'fr' : 'en';

  return (
    <SegmentedControl
      size="xs"
      radius="xl"
      value={lang}
      onChange={(value) => i18n.changeLanguage(value)}
      data={[
        { label: 'FR', value: 'fr' },
        { label: 'EN', value: 'en' },
      ]}
      style={{
        position: 'fixed',
        top: 'calc(env(safe-area-inset-top, 0px) + 10px)',
        right: 'calc(env(safe-area-inset-right, 0px) + 10px)',
        zIndex: 200,
      }}
    />
  );
}
