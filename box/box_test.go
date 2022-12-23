/*
Package box contains the network and business logic of virtual-säemubox.

Copyright © 2020 Radio Bern RaBe - Lucas Bickel <hairmare@rabe.ch>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package box

import (
	"testing"
)

func Test_checkTrimmedData(t *testing.T) {
	type args struct {
		trimmedData string
	}
	tests := []struct {
		name         string
		args         args
		wantTarget   int32
		wantOnChange bool
		wantErr      bool
	}{
		{
			name: "login sucessful",
			args: args{
				trimmedData: "login successful",
			},
			wantTarget:   0,
			wantOnChange: false,
		},
		{
			name: "login failed",
			args: args{
				trimmedData: "login failed",
			},
			wantTarget:   0,
			wantOnChange: false,
			wantErr:      true,
		},
		{
			name: "klangbecken is active lowercase pinstate",
			args: args{
				trimmedData: "PinState=l",
			},
			wantTarget:   1,
			wantOnChange: true,
		},
		{
			name: "klangbecken is active uppercase pinstate",
			args: args{
				trimmedData: "PinState=L",
			},
			wantTarget:   1,
			wantOnChange: true,
		},
		{
			name: "default to studio",
			args: args{
				trimmedData: "PinState=h",
			},
			wantTarget:   6,
			wantOnChange: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTarget, gotOnChange, err := checkTrimmedData(tt.args.trimmedData)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkTrimmedData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotTarget != tt.wantTarget {
				t.Errorf("checkTrimmedData() gotTarget = %v, want %v", gotTarget, tt.wantTarget)
			}
			if gotOnChange != tt.wantOnChange {
				t.Errorf("checkTrimmedData() gotOnChange = %v, want %v", gotOnChange, tt.wantOnChange)
			}
		})
	}
}
