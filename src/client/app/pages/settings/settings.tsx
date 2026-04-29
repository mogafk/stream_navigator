import { PageContainer } from '~/components/PageContainer/page-container'
import { ConfigList } from '~/components/ConfigList/config-list'
import "./settings.css";

export function Settings() {
  return (
    <PageContainer>
      <div className="content">
        <div className="content_left">
          <ConfigList configList={['default.json', 'wow.json']} />
        </div>
        <div>
          content
        </div>
        <div>
          right menu
        </div>
      </div>
    </PageContainer>

  );
}

