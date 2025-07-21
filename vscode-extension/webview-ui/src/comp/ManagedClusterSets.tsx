import { VSCodeDataGrid, VSCodeDataGridCell, VSCodeDataGridRow,  } from '@vscode/webview-ui-toolkit/react';
import { useState, useEffect } from 'react';
import { OcmResource } from '../../../src/data/loader'

export default function ShowManagedClusterSets() {
    let [managedClusterSets, setManagedClusterSets] = useState<OcmResource[]>([]);

	useEffect(() => {
        window.addEventListener("message", event => {
			if ('crsDistribution' in event.data && 'ManagedClusterSet' === event.data.crsDistribution.kind) {
				setManagedClusterSets(JSON.parse(event.data.crsDistribution.crs));
			}
        });
    });

    return (
        <section className="component-row">
            { managedClusterSets.length > 0 &&
                <>
                    <h2 style={{ marginTop: '40px' }}>Managed Cluster Sets</h2>
                    <VSCodeDataGrid gridTemplateColumns="1fr 1fr" aria-label='ManagedClusterSets' >
                        <VSCodeDataGridRow rowType="sticky-header">
                        <VSCodeDataGridCell cellType='columnheader' gridColumn='1'>Managed Cluster Set Name</VSCodeDataGridCell>
                        <VSCodeDataGridCell cellType='columnheader' gridColumn='2'>Conditions</VSCodeDataGridCell>
                    </VSCodeDataGridRow>

                    {managedClusterSets.map(managedClusterSet => {
                        return <VSCodeDataGridRow>
                                    <VSCodeDataGridCell gridColumn='1'>{managedClusterSet.kr.metadata.name}</VSCodeDataGridCell>
                                    <VSCodeDataGridCell gridColumn='2'>{managedClusterSet.kr.status.conditions.map( ( condition:any )=> { return<p> - lastTransitionTime: {condition.lastTransitionTime}, message: {condition.message}, reason: {condition.reason}, status: {condition.status}, type: {condition.type} </p>  })} </VSCodeDataGridCell>
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
