import { VSCodeDataGrid, VSCodeDataGridCell, VSCodeDataGridRow,  } from '@vscode/webview-ui-toolkit/react';
import { useState, useEffect } from 'react';
import { OcmResource } from '../../../src/data/loader'

export default  function ShowAppliedManifestWorks() {
    let [appliedManifestWorks, setAppliedManifestWorks] = useState<OcmResource[]>([]);

	useEffect(() => {
		window.addEventListener("message", event => {
			if ('crsDistribution' in event.data && 'AppliedManifestWork' === event.data.crsDistribution.kind) {
				setAppliedManifestWorks(JSON.parse(event.data.crsDistribution.crs));
			}
        });
    });

    return (
        <section className="component-row">
            { appliedManifestWorks.length > 0 &&
                <>
                    <h2 style={{ marginTop: '40px' }}>Applied ManifestWorks</h2>
                    <VSCodeDataGrid gridTemplateColumns="1fr 1fr 2fr" aria-label='AppliedManifestWork' >
                        <VSCodeDataGridRow rowType="sticky-header">
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='1'>ManifestWork Name</VSCodeDataGridCell>
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='2'>Creation TimeStamp</VSCodeDataGridCell>
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='3'>Applied Resources</VSCodeDataGridCell>
                        </VSCodeDataGridRow>

                        {appliedManifestWorks.map(appliedManifestWork => {
                            return <VSCodeDataGridRow>
                                        <VSCodeDataGridCell gridColumn='1'>{appliedManifestWork.kr.spec.manifestWorkName}</VSCodeDataGridCell>
                                        <VSCodeDataGridCell gridColumn='2'>{appliedManifestWork.kr.metadata.creationTimestamp} </VSCodeDataGridCell>
                                        <VSCodeDataGridCell gridColumn='3'>{appliedManifestWork.kr.status.appliedResources.map( ( resource:any )=> { return<p>  - group: {resource.group}, name: {resource.name}, namespace: {resource.namespace}, resource: {resource.resource} </p>  })} </VSCodeDataGridCell>
                                    </VSCodeDataGridRow>
                        } )
                        }
                    </VSCodeDataGrid>
                    <div style={{ borderTop: "1px solid #fff ", marginLeft: 10, marginRight: 10 }}></div>
                </>
            }
        </section>
    );
}
