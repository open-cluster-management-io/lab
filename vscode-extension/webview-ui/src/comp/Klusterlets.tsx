import { VSCodeDataGrid, VSCodeDataGridCell, VSCodeDataGridRow,  } from '@vscode/webview-ui-toolkit/react';
import { useState, useEffect } from 'react';
import { OcmResource } from '../../../src/data/loader'

export default function ShowKlusterlets() {
    let [klusterlets, setKlusterlets] = useState<OcmResource[]>([]);

	useEffect(() => {
        window.addEventListener("message", event => {
			if ('crsDistribution' in event.data && 'Klusterlet' === event.data.crsDistribution.kind) {
				setKlusterlets(JSON.parse(event.data.crsDistribution.crs));
			}
        });
    });

    return (
        <section className="component-row">
            { klusterlets.length > 0 &&
                <>
                    <h2 style={{ marginTop: '40px' }}>Klusterlet</h2>
                    <VSCodeDataGrid gridTemplateColumns="1fr 1fr" aria-label='Klusterlet' >
                        <VSCodeDataGridRow rowType="sticky-header">
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='1'>Klusterlet Name</VSCodeDataGridCell>
                                <VSCodeDataGridCell cellType='columnheader' gridColumn='2'>Conditions</VSCodeDataGridCell>
                        </VSCodeDataGridRow>

                        {klusterlets.map(klusterlet => {
                            return <VSCodeDataGridRow>
                                        <VSCodeDataGridCell gridColumn='1'>{klusterlet.kr.metadata.name}</VSCodeDataGridCell>
                                        <VSCodeDataGridCell gridColumn='2'>{klusterlet.kr.status.conditions.map( ( condition:any )=> { return<p> - lastTransitionTime: {condition.lastTransitionTime}, message: {condition.message}, reason: {condition.reason}, status: {condition.status}, type: {condition.type} </p>  })} </VSCodeDataGridCell>
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
