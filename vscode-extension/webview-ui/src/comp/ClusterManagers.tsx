import { VSCodeDataGrid, VSCodeDataGridCell, VSCodeDataGridRow,  } from '@vscode/webview-ui-toolkit/react';
import { useState, useEffect } from 'react';
import { OcmResource } from '../../../src/data/loader'

export default function ShowClusterManagers() {
    let [clusterManagers, setClusterManagers] = useState<OcmResource[]>([]);

	useEffect(() => {
        window.addEventListener("message", event => {
			if ('crsDistribution' in event.data && 'ClusterManager' === event.data.crsDistribution.kind) {
				setClusterManagers(JSON.parse(event.data.crsDistribution.crs));
			}
		});
    });

    return (
        <section className="component-row">
            { clusterManagers.length > 0 &&
                <>
                    <h2 style={{ marginTop: '40px' }}>Cluster Manager</h2>
                    <VSCodeDataGrid gridTemplateColumns="1fr 1fr" aria-label='ClusterManager' >
                        <VSCodeDataGridRow rowType="sticky-header">
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='1'>Cluster Manager Name</VSCodeDataGridCell>
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='2'>Conditions</VSCodeDataGridCell>
                        </VSCodeDataGridRow>

                        {clusterManagers.map(clusterManager => {
                            return <VSCodeDataGridRow>
                                        <VSCodeDataGridCell gridColumn='1'>{clusterManager.kr.metadata.name}</VSCodeDataGridCell>
                                        <VSCodeDataGridCell gridColumn='2'>{clusterManager.kr.status.conditions.map( ( condition:any )=> { return<p> - lastTransitionTime: {condition.lastTransitionTime}, message: {condition.message}, reason: {condition.reason}, status: {condition.status}, type: {condition.type} </p>  })} </VSCodeDataGridCell>
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
